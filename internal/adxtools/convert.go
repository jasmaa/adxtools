// Functions for converting

package adxtools

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/youpy/go-wav"
)

func Adx2Wav(fname string) {

	// Open wav to write to
	inFile, err := os.Create(fname)
	if err != nil {
		fmt.Println(err)
	}

	// Open adx
	outFile, err := os.Open("BGM_002.adx")
	if err != nil {
		fmt.Println(err)
	}

	defer inFile.Close()
	defer outFile.Close()

	writer := wav.NewWriter(inFile, 90000, 2, 44100, 16)

	// Decode ADX header
	adx := ADX{}
	adx.ReadHeader("BGM_002.adx")

	// Calculate prediction coefficients
	a := math.Sqrt(2) - math.Cos(2)*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate)
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
	samplesNeeded := uint(30000)
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
		start := (uint(adx.copyrightOffset) + 4 + uint(sampleIndex)/samplesPerBlock*uint(adx.blockSize)*uint(adx.channelCount)) // fix me

		for i := uint8(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.ReadAt(scaleBytes, int64(start+uint(adx.blockSize*i)))
			scale[i] = int16(binary.BigEndian.Uint16(scaleBytes)) // fix me
		}

		sampleEndOffset := sampleOffset + samplesCanGet

		start += 2
		for sampleOffset < sampleEndOffset {
			fmt.Printf("SAMPLE: %d\n", sampleIndex)

			for i := uint8(0); i < adx.channelCount; i++ {
				samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])

				sampleErrorBytes := make([]byte, adx.sampleBitdepth)
				inFile.ReadAt(sampleErrorBytes, int64(start+(uint(adx.sampleBitdepth)*sampleOffset)/8+uint(adx.blockSize*i)))
				sampleError := int32(binary.BigEndian.Uint32(sampleErrorBytes)) // fix me
				// SIGN EXTEND????
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
				fmt.Printf("%x %x\n", start, sampleOffset)
				fmt.Printf("%d\n", sampleErrorBytes)

				buffer := make([]wav.Sample, 1)
				buffer[0] = wav.Sample{[2]int{int(sample), int(sample)}}
				writer.WriteSamples(buffer)
			}

			sampleOffset++
			sampleIndex++
			samplesNeeded--
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
