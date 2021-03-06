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
func Wav2Adx(inPath string, outPath string,
	highpassFrequency uint16, loopBeginSampleIndex uint32, loopEndSampleIndex uint32) {

	startTime := time.Now()

	// Open files
	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}

	defer outFile.Close()

	inFile, err := os.Open(inPath)
	if err != nil {
		panic(err)
	}

	defer inFile.Close()

	reader := wav.NewReader(inFile)
	format, err := reader.Format()
	if err != nil {
		panic(err)
	}

	// Encode ADX header
	adx := header{
		copyrightOffset:      404, // arbitrary offset
		encodingType:         0x03,
		blockSize:            18,
		sampleBitdepth:       4,
		channelCount:         byte(format.NumChannels),
		sampleRate:           format.SampleRate,
		highpassFrequency:    highpassFrequency,
		version:              4,
		flags:                0,
		loopAlignmentSamples: 0,
		loopEnabled:          loopBeginSampleIndex < loopEndSampleIndex,
		loopBeginSampleIndex: loopBeginSampleIndex,
		loopEndSampleIndex:   loopEndSampleIndex,
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

	// Determine looping bytes
	adx.SetLoopBytes(samplesPerBlock)

	// Loop per block until EOF
	for {

		buffer, err := reader.ReadSamples(uint32(samplesPerBlock)) // Read in a frame of samples
		if err != nil {
			break
		}

		// Get offset and start position
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		scaledSampleErrorNibbles := make([]int32, uint32(adx.channelCount)*uint32(samplesPerBlock)) // convert to be pedantic

		samplesCanGet := samplesPerBlock
		if byte(len(buffer)) < samplesPerBlock {
			samplesCanGet = byte(len(buffer))
		}

		// Encode samples
		for sampleOffset := byte(0); sampleOffset < samplesCanGet; sampleOffset++ {

			inSamples := buffer[sampleOffset].Values

			// Process for each channel in sample
			for i := byte(0); i < adx.channelCount; i++ {

				samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])
				sample := int32(inSamples[i])

				scaledSampleErrorNibbles[samplesPerBlock*byte(i)+sampleOffset] = sample - int32(samplePrediction)

				// Update past samples
				pastSamples[i*2+1] = pastSamples[i*2+0]
				pastSamples[i*2+0] = sample
			}

			sampleIndex++
		}

		// Generate scale and sample error bytes
		scale := generateScale(&adx, samplesPerBlock, scaledSampleErrorNibbles)
		sampleErrorBytes := generateSampleError(&adx, samplesPerBlock, scaledSampleErrorNibbles, scale)

		// Write block
		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(scaleBytes, scale[i])
			outFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			outFile.Write(scaleBytes)

			sectionLen := len(sampleErrorBytes) / int(adx.channelCount)
			outFile.Seek(int64(start+2+uint32(adx.blockSize)*uint32(i)), 0)
			outFile.Write(sampleErrorBytes[sectionLen*int(i) : sectionLen*int(i+1)])
		}
	}

	// Write metadata
	adx.totalSamples = sampleIndex
	adx.Write(outFile)

	outFile.Seek(int64(adx.copyrightOffset-2), 0)
	outFile.Write([]byte("(c)CRI"))

	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}

// Generates scale based on unscaled samples
func generateScale(adx *header, samplesPerBlock byte, scaledSampleErrorNibbles []int32) []uint16 {

	scale := make([]uint16, adx.channelCount)

	for i := byte(0); i < adx.channelCount; i++ {

		// Get max reach
		minAbsErr := scaledSampleErrorNibbles[samplesPerBlock*i+0]
		maxAbsErr := scaledSampleErrorNibbles[samplesPerBlock*i+0]
		for j := byte(0); j < samplesPerBlock; j++ {

			v := scaledSampleErrorNibbles[samplesPerBlock*i+j]

			if v > maxAbsErr {
				maxAbsErr = v
			}
			if v < minAbsErr {
				minAbsErr = v
			}
		}

		// Calculate scale
		if maxAbsErr > 0 && minAbsErr < 0 {
			if maxAbsErr > -minAbsErr {
				scale[i] = uint16(maxAbsErr / 7)
			} else {
				scale[i] = uint16(minAbsErr / -8)
			}
		} else if minAbsErr > 0 {
			scale[i] = uint16(maxAbsErr / 7)
		} else if maxAbsErr < 0 {
			scale[i] = uint16(minAbsErr / -8)
		}
	}

	return scale
}

// Unscales and merges error bytes
func generateSampleError(adx *header, samplesPerBlock byte, scaledSampleErrorNibbles []int32, scale []uint16) []byte {

	// Unscale to 4-bit bitdepth
	sampleErrorNibbles := make([]byte, len(scaledSampleErrorNibbles))
	for i := byte(0); i < adx.channelCount; i++ {
		for j := byte(0); j < samplesPerBlock; j++ {

			unscaledError := byte(0)
			if scale[i] != 0 {

				unscaledError = byte(scaledSampleErrorNibbles[samplesPerBlock*i+j] / int32(scale[i]))
			}

			sampleErrorNibbles[samplesPerBlock*i+j] = unscaledError
		}
	}

	// Merge nibbles
	// TODO: make it work with odd??
	sampleErrorBytes := make([]byte, len(sampleErrorNibbles)/2)
	for i := 0; i < len(sampleErrorNibbles); i += 2 {

		sampleErrorBytes[i/2] = (sampleErrorNibbles[i+0] << 4) | (sampleErrorNibbles[i+1] & 0xF)
	}

	return sampleErrorBytes
}
