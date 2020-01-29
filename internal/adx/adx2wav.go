package adx

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/youpy/go-wav"
)

// Adx2Wav converts ADX input to WAV output ignoring loops
// TODO: optimize
func Adx2Wav(inPath string, outPath string) {

	startTime := time.Now()

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
	adx.Read(inFile)

	writer := wav.NewWriter(outFile, uint32(adx.totalSamples), uint16(adx.channelCount), uint32(adx.sampleRate), 16)

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
	scale := make([]uint16, adx.channelCount)

	//fmt.Printf("%+v\n", adx)
	//fmt.Println(index2byte(&adx, samplesPerBlock, adx.loopBeginSampleIndex))
	//fmt.Println(index2byte(&adx, samplesPerBlock, adx.loopEndSampleIndex))

	// Loop per block until EOF
	for sampleIndex < adx.totalSamples {

		// Get offset and start position
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		buffer := make([]byte, uint32(adx.blockSize)*uint32(adx.channelCount))
		inFile.Seek(int64(start), 0)
		inFile.Read(buffer)

		outBuffer := make([]wav.Sample, samplesPerBlock)

		// Read scale
		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			inFile.Read(scaleBytes)
			scale[i] = binary.BigEndian.Uint16(scaleBytes)
		}

		// Decode samples
		for sampleOffset := 0; sampleOffset < int(samplesPerBlock); sampleOffset += 2 {

			outSamples := make([]int, 2*adx.channelCount)

			// Process for each channel in sample
			for i := byte(0); i < adx.channelCount; i++ {

				// HARD CODE: Read in sample with 4-bit bitdepth
				sampleErrorNibbles := make([]byte, 2)
				sampleErrorNibbles[0] = buffer[uint32(adx.blockSize)*uint32(i)+2+uint32(sampleOffset)/2] >> 4
				sampleErrorNibbles[1] = buffer[uint32(adx.blockSize)*uint32(i)+2+uint32(sampleOffset)/2] & 0xF

				for nibbleIdx, v := range sampleErrorNibbles {

					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

					// Sign extend 4-bit to 32-bit signed int
					sampleError := int32(v)
					sampleError = (sampleError << 28) >> 28

					sampleError *= int32(scale[i])
					sample := sampleError + int32(samplePrediction)

					// Update past samples
					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample

					// Clamp sample within 16-bit bit depth range
					if sample > 32767 {
						sample = 32767
					} else if sample < -32768 {
						sample = -32768
					}

					outSamples[int(i)*2+nibbleIdx] = int(sample)
				}
			}

			switch adx.channelCount {

			case 1:
				outBuffer[sampleOffset+0] = wav.Sample{[2]int{outSamples[0], 0}}
				outBuffer[sampleOffset+1] = wav.Sample{[2]int{outSamples[1], 0}}
			case 2:
				outBuffer[sampleOffset+0] = wav.Sample{[2]int{outSamples[0], outSamples[2]}}
				outBuffer[sampleOffset+1] = wav.Sample{[2]int{outSamples[1], outSamples[3]}}
			}

			sampleIndex += 2
		}

		// Write to wav
		writer.WriteSamples(outBuffer)
	}

	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}
