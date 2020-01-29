package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jasmaa/adxtools/internal/adx"
)

const usage = `Usage: adxtools [COMMAND] [OPTIONS] input
Commands:
	- adx2wav
		Convert ADX to WAV

	- wav2adx
		Convert WAV to ADX`

func main() {

	adx2wavCmd := flag.NewFlagSet("adx2wav", flag.ExitOnError)
	wavOut := adx2wavCmd.String("o", "out.wav", "Output WAV file name")

	wav2adxCmd := flag.NewFlagSet("wav2adx", flag.ExitOnError)
	adxOut := wav2adxCmd.String("o", "out.adx", "Output ADX file name")
	highpassFrequency := wav2adxCmd.Uint("highpass", 2000, "Highpass frequency")
	loopBeginSampleIndex := wav2adxCmd.Uint("loopBegin", 0, "Loop begin sample index")
	loopEndSampleIndex := wav2adxCmd.Uint("loopEnd", 0, "Loop end sample index")

	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	switch os.Args[1] {

	case "adx2wav":
		adx2wavCmd.Parse(os.Args[2:])
		rest := adx2wavCmd.Args()

		if len(rest) < 1 {
			adx2wavCmd.Usage()
			os.Exit(1)
		}

		adx.Adx2Wav(rest[0], *wavOut)

	case "wav2adx":
		wav2adxCmd.Parse(os.Args[2:])
		rest := wav2adxCmd.Args()

		if len(rest) < 1 {
			wav2adxCmd.Usage()
			os.Exit(1)
		}

		adx.Wav2Adx(rest[0], *adxOut, uint16(*highpassFrequency), uint32(*loopBeginSampleIndex), uint32(*loopEndSampleIndex))

	default:
		fmt.Println(usage)
	}
}
