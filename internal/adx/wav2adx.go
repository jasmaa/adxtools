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
		totalSamples:         uint32(90000), //temp
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
	adx.Write(outPath)

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

	// Make scale
	scale := make([]int32, adx.channelCount)
	for i := range scale {
		scale[i] = 1
	}

	// Loop per block until EOF
	for {

		bufferCount := 0
		buffer, err := reader.ReadSamples(uint32(samplesPerBlock))
		if err != nil {
			break
		}

		samplesCanWrite := byte(len(buffer))

		// Get offset and start position
		sampleOffset := sampleIndex % uint32(samplesPerBlock)
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		unscaledSampleErrorBytes := make([]int32, uint32(adx.channelCount)*uint32(samplesCanWrite)) // convert to be pedantic

		// Encode samples
		start += 2
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

					unscaledSampleErrorBytes[i*offset+i] = unscaledSampleError
					/*
						// Get nibble
						sampleErrorNibbles[i] = byte(sampleError)
					*/

					// Update past samples
					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample
				}

				/*
					sampleErrorBytes := []byte{(sampleErrorNibbles[0] << 4) | (sampleErrorNibbles[1] & 0xF)}

					inFile.Seek(int64(start+(uint32(adx.sampleBitdepth)*sampleOffset)/8+uint32(adx.blockSize)*uint32(i)), 0)
					inFile.Write(sampleErrorBytes)
				*/
			}

			sampleOffset += 2
			sampleIndex += 2
		}

		// TODO: generate sampleError and scale here

		// Write block
		//inFile.Seek(int64(start+(uint32(adx.sampleBitdepth)*sampleOffset)/8+uint32(adx.blockSize)*uint32(i)), 0)
		//inFile.Write(sampleErrorBytes)

		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(scaleBytes, uint16(scale[i]))
			inFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			inFile.Write(scaleBytes)
		}
	}
	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}

func calculateScale(unscaledSampleErrorBytes *[]int32) {
}
