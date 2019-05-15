// Functions for converting

package adxtools

import (
	"fmt"
	"io"
	"os"

	"github.com/youpy/go-wav"
)

func Adx2Wav(fname string) {

	/*
		// Open wav to write to
		f, err := os.Create(fname)
		if err != nil {
			fmt.Println(err)
		}

		defer f.Close()

		writer := wav.NewWriter(f, 2000, 2, 44100, 16)
	*/

	// Decode ADX
	adx := ADX{}
	adx.ReadHeader("BGM_002.adx")

	fmt.Printf("%v", adx)
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
