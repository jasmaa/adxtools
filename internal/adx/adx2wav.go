package adx

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/youpy/go-wav"
)

// Adx2Wav converts ADX input to WAV output ignoring loops
func Adx2Wav(inPath string, outPath string) {

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
	adx.Read(inFile)

	writer := wav.NewWriter(outFile, uint32(adx.totalSamples), 2, uint32(adx.sampleRate), 16)

	// Calculate prediction coefficients and init structs
	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint32(0)

	samplesNeeded := adx.totalSamples // TODO: Make sure sample number is even?
	samplesPerBlock := (adx.blockSize - 2) * 8 / adx.sampleBitdepth
	scale := make([]int32, adx.channelCount)

	// Loop until got number of samples needed or EOF
	for samplesNeeded > 0 && sampleIndex < adx.totalSamples {

		// Calculate number of samples left
		sampleOffset := sampleIndex % uint32(samplesPerBlock)
		samplesCanGet := uint32(samplesPerBlock) - sampleOffset

		// Get start offset and scale
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			inFile.Read(scaleBytes)
			scale[i] = int32(binary.BigEndian.Uint16(scaleBytes))
		}

		sampleEndOffset := sampleOffset + samplesCanGet

		bufferCount := 0
		buffer := make([]wav.Sample, samplesCanGet)

		start += 2

		// Decode samples
		for sampleOffset < sampleEndOffset {

			outSamples := make([]int, 2*adx.channelCount)

			for i := byte(0); i < adx.channelCount; i++ {

				// HARD CODE: BITDEPTH 4, do two samples at once
				sampleErrorBytes := make([]byte, 1)
				inFile.Seek(int64(start+(uint32(adx.sampleBitdepth)*sampleOffset)/8+uint32(adx.blockSize)*uint32(i)), 0)
				inFile.Read(sampleErrorBytes)

				sampleErrorNibbles := make([]byte, 2)
				sampleErrorNibbles[0] = sampleErrorBytes[0] >> 4
				sampleErrorNibbles[1] = sampleErrorBytes[0] & 0xF

				for nibbleIdx, v := range sampleErrorNibbles {

					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

					// Sign extend 4-bit to 32-bit signed int
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

					outSamples[int(i)*2+nibbleIdx] = int(sample)
				}
			}

			// HARD CODE: Write to 2 channel buffer
			buffer[bufferCount+0] = wav.Sample{[2]int{outSamples[0], outSamples[2]}}
			buffer[bufferCount+1] = wav.Sample{[2]int{outSamples[1], outSamples[3]}}
			bufferCount += 2

			sampleOffset += 2
			sampleIndex += 2
			samplesNeeded -= 2
		}

		writer.WriteSamples(buffer)
	}
	fmt.Printf("Elapsed: %v seconds", time.Now().Sub(startTime).Seconds())
}

// Read in wav samples
func readWavSamples(fname string) {

	file, err := os.Open(fname)
	if err != nil {
		fmt.Println(err)
	}

	reader := wav.NewReader(file)

	defer file.Close()

	// Read samples
	var count = 0
	for {
		samples, err := reader.ReadSamples()
		if err == io.EOF {
			break
		}

		for _, sample := range samples {
			fmt.Printf("L/R: %d/%d\n", reader.IntValue(sample, 0), reader.IntValue(sample, 1))
			count++
		}
	}

	fmt.Printf("%d", count)
}

func writeWavSamples(fname string) {
	file, err := os.Create(fname)
	if err != nil {
		fmt.Println(err)
	}

	defer file.Close()

	writer := wav.NewWriter(file, 2000, 2, 44100, 16)

	samples := make([]wav.Sample, 16)
	samples[0] = wav.Sample{[2]int{3000, 4000}}

	samples2 := make([]wav.Sample, 16)
	samples2[0] = wav.Sample{[2]int{8000, 0}}

	writer.WriteSamples(samples)
	writer.WriteSamples(samples2)

	fmt.Println("Wrote samples: %v", samples)
}
