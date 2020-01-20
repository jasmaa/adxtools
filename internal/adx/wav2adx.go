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

	// Decode ADX header
	adx := header{}
	adx.Read(inPath)

	writer := wav.NewWriter(outFile, uint32(adx.totalSamples), 2, uint32(adx.sampleRate), 16)

	// Calculate prediction coefficients and init structs
	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint(0)

	samplesNeeded := adx.totalSamples // TODO: Make sure sample number is even?
	samplesPerBlock := (adx.blockSize - 2) * 8 / adx.sampleBitdepth
	scale := make([]int32, adx.channelCount)

	// Loop until got number of samples needed or EOF
	for samplesNeeded > 0 && sampleIndex < adx.totalSamples {

		// Calculate number of samples left
		sampleOffset := sampleIndex % samplesPerBlock
		samplesCanGet := samplesPerBlock - sampleOffset

		// Get start offset and scale
		start := adx.copyrightOffset + 4 + sampleIndex/samplesPerBlock*adx.blockSize*adx.channelCount

		for i := uint(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.Seek(int64(start+adx.blockSize*i), 0)
			inFile.Read(scaleBytes)
			scale[i] = int32(binary.BigEndian.Uint16(scaleBytes))
		}

		sampleEndOffset := sampleOffset + samplesCanGet

		bufferCount := 0
		buffer := make([]wav.Sample, samplesCanGet)

		// Decode samples
		start += 2
		for sampleOffset < sampleEndOffset {

			outSamples := make([]int, adx.channelCount)

			for i := uint(0); i < adx.channelCount; i++ {

				// HARD CODE: BITDEPTH 4, do two samples at once
				sampleErrorBytes := make([]byte, 1)
				inFile.Seek(int64(start+(adx.sampleBitdepth*sampleOffset)/8+adx.blockSize*i), 0)
				inFile.Read(sampleErrorBytes)

				sampleErrorNibbles := make([]byte, 2)
				sampleErrorNibbles[0] = sampleErrorBytes[0] >> 4
				sampleErrorNibbles[1] = sampleErrorBytes[0] & 0xF

				for _, v := range sampleErrorNibbles {

					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

					// Sign extend 28-bit to 32-bit signed int
					sampleError := int32(v)
					sampleError = (sampleError << 28) >> 28

					sampleError *= scale[i]

					sample := sampleError + int32(samplePrediction)

					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample

					// Clamp sample within 16-bit bit depth range
					if sample > 32767 {
						sample = 32767
					} else if sample < -32768 {
						sample = -32768
					}

					outSamples[i] = int(sample)
				}

				// HARD CODE: Write to 2 channel buffer
				buffer[bufferCount] = wav.Sample{[2]int{outSamples[0], outSamples[1]}}
				bufferCount++
			}

			sampleOffset += 2
			sampleIndex += 2
			samplesNeeded -= 2
		}

		writer.WriteSamples(buffer)
	}
	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}
