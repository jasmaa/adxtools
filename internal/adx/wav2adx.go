package adx

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"

	wav "github.com/youpy/go-wav"
)

// Wav2Adx converts WAV input to ADX output
func Wav2Adx(inPath string, outPath string) {

	startTime := time.Now()

	// Open files
	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Println(err)
	}

	defer outFile.Close()

	inFile, err := os.Open(inPath)
	if err != nil {
		fmt.Println(err)
	}

	defer inFile.Close()

	reader := wav.NewReader(inFile)
	format, err := reader.Format()
	if err != nil {
		panic(err)
	}

	// Encode ADX header
	adx := header{
		copyrightOffset:      404, //???
		encodingType:         0x03,
		blockSize:            18,
		sampleBitdepth:       4,
		channelCount:         byte(format.NumChannels),
		sampleRate:           format.SampleRate,
		highpassFrequency:    500,
		version:              3,
		flags:                0,
		loopAlignmentSamples: 0,
		loopEnabled:          false,

		//TODO: Figure looping out
		loopBeginSampleIndex: 0,
		loopBeginByteIndex:   0,
		loopEndSampleIndex:   0,
		loopEndByteIndex:     0,
	}

	// Calculate prediction coefficients and init structs
	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint32(0)

	samplesPerBlock := (adx.blockSize - 2) * 8 / adx.sampleBitdepth

	// Loop per block until EOF
	for {

		bufferCount := 0
		buffer, err := reader.ReadSamples(uint32(samplesPerBlock))
		if err != nil {
			break
		}

		samplesCanWrite := byte(len(buffer))

		// Get offset and start position
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		unscaledSampleErrorNibbles := make([]int32, uint32(adx.channelCount)*uint32(samplesCanWrite)) // convert to be pedantic

		// Encode samples
		for offset := byte(0); offset < samplesCanWrite; offset++ {

			// HARD CODE: Write to 2 channel buffer
			inSamples := buffer[bufferCount].Values
			bufferCount++

			for i := byte(0); i < adx.channelCount; i++ {

				//sampleErrorNibbles := make([]byte, 2)

				for _, v := range inSamples {

					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

					sample := int32(v)

					// Clamp sample within 16-bit bit depth range
					if sample > 32767 {
						sample = 32767
					} else if sample < -32768 {
						sample = -32768
					}

					unscaledSampleError := sample - int32(samplePrediction)

					unscaledSampleErrorNibbles[i*offset+i] = unscaledSampleError

					// Update past samples
					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample
				}
			}

			sampleIndex++
		}

		// Generate scale and sample error bytes
		scale := generateScale(&adx, unscaledSampleErrorNibbles)
		sampleErrorBytes := generateSampleError(&adx, unscaledSampleErrorNibbles, scale)

		// Write block
		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(scaleBytes, scale[i])
			outFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			outFile.Write(scaleBytes)
		}

		start += 2

		outFile.Seek(int64(start), 0)
		outFile.Write(sampleErrorBytes)

		//fmt.Println(len(sampleErrorBytes))
	}

	// Write metadata
	adx.totalSamples = sampleIndex
	adx.Write(outFile)

	outFile.Seek(int64(adx.copyrightOffset-2), 0)
	outFile.Write([]byte("(c)CRI"))

	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}

// Generates scale based on unscaled samples
func generateScale(adx *header, unscaledSampleErrorBytes []int32) []uint16 {

	scale := make([]uint16, adx.channelCount)

	for i := byte(0); i < adx.channelCount; i++ {

		// Get max reach
		maxAbsErr := unscaledSampleErrorBytes[i]
		for j := byte(0); j < byte(len(unscaledSampleErrorBytes))/adx.channelCount; j++ {

			v := unscaledSampleErrorBytes[j*adx.channelCount+i]
			if v < 0 {
				v = -v
			}

			if v > maxAbsErr {
				maxAbsErr = v
			}
		}

		// Calculate scale
		scale[i] = uint16(maxAbsErr / 8)
		if scale[i] == 0 {
			scale[i] = 1
		}
	}

	return scale
}

// Scales error bytes
func generateSampleError(adx *header, unscaledSampleErrorBytes []int32, scale []uint16) []byte {

	sampleErrorNibbles := make([]byte, len(unscaledSampleErrorBytes))

	// Scale to 4-bit bitdepth
	for i := byte(0); i < adx.channelCount; i++ {
		for j := byte(0); j < byte(len(unscaledSampleErrorBytes))/adx.channelCount; j++ {

			sampleErrorNibbles[j*adx.channelCount+i] = byte(unscaledSampleErrorBytes[j*adx.channelCount+i] / int32(scale[i]))
		}
	}

	// Merge nibbles
	// TODO: make it work with odd??
	sampleErrorBytes := make([]byte, len(unscaledSampleErrorBytes)/int(adx.channelCount))
	count := 0
	for i := 0; i < len(sampleErrorBytes); i += 2 {
		sampleErrorBytes[count] = (sampleErrorNibbles[i+0] << 4) | (sampleErrorNibbles[i+1] & 0xF)
		count++
	}

	return sampleErrorBytes
}
