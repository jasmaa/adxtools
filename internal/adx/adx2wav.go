package adx

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/youpy/go-wav"
)

func Adx2Wav(inputFile string, outputFile string) {

	// Open wav to write to
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Println(err)
	}

	// Open adx
	inFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Println(err)
	}

	defer inFile.Close()
	defer outFile.Close()

	// Decode ADX header
	adx := header{}
	adx.Read(inputFile)

	writer := wav.NewWriter(outFile, adx.totalSamples, 2, adx.sampleRate, 16)

	// Calculate prediction coefficients
	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint32(0)

	// =========================
	// DECODE ADX IGNORING LOOPS
	// =========================
	samplesNeeded := uint(adx.totalSamples)
	samplesPerBlock := uint((adx.blockSize - 2) * 8 / adx.sampleBitdepth)
	scale := make([]int16, adx.channelCount)

	// Loop until got number of samples needed or EOF
	for samplesNeeded > 0 && sampleIndex < adx.totalSamples {

		// Calculate number of samples left
		sampleOffset := uint(sampleIndex) % samplesPerBlock
		samplesCanGet := samplesPerBlock - sampleOffset

		// Clamp samplesCanGet
		if samplesCanGet > samplesNeeded {
			samplesCanGet = samplesNeeded
		}

		// Calculate start offset in bytes
		start := uint(adx.copyrightOffset) + 4 + uint(sampleIndex)/samplesPerBlock*uint(adx.blockSize)*uint(adx.channelCount)

		for i := uint8(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.ReadAt(scaleBytes, int64(start+uint(adx.blockSize*i)))
			scale[i] = int16(binary.BigEndian.Uint16(scaleBytes))
		}

		sampleEndOffset := sampleOffset + samplesCanGet

		start += 2
		for sampleOffset < sampleEndOffset {
			fmt.Printf("SAMPLE: %d\n", sampleIndex)

			outSamples := make([]int, adx.channelCount)

			for i := uint8(0); i < adx.channelCount; i++ {
				// HARD CODE BITDEPTH 4
				sampleErrorBytes := make([]byte, 1)
				inFile.ReadAt(sampleErrorBytes, int64(start+(uint(adx.sampleBitdepth)*sampleOffset)/8+uint(adx.blockSize*i)))

				sampleErrorNibbles := make([]byte, 2)
				sampleErrorNibbles[0] = sampleErrorBytes[0] >> 4
				sampleErrorNibbles[1] = sampleErrorBytes[0] & 0xF

				for _, v := range sampleErrorNibbles {
					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

					// sign extend
					sampleError := int32(v)
					sampleError = (sampleError << 28) >> 28

					sampleError *= int32(scale[i])

					sample := sampleError + int32(samplePrediction)

					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample

					if sample > 32767 {
						sample = 32767
					} else if sample < -32768 {
						sample = -32768
					}

					// write buffer immediately for now
					//fmt.Printf("%x %x\n", start, sampleOffset)
					fmt.Printf("%d\n", sample)
					outSamples[i] = int(sample)
				}

				// HARDCODE WRITE to 2CHANNEL BUFFER
				buffer := make([]wav.Sample, 1)
				buffer[0] = wav.Sample{[2]int{outSamples[0], outSamples[1]}}
				writer.WriteSamples(buffer)
			}

			sampleOffset += 2
			sampleIndex += 2
			samplesNeeded -= 2
		}
	}
}

func Wav2Adx(fname string) {
	return
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
