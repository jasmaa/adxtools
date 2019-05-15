// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jasmaa/adxtools/internal/adxtools"
)

// wav2adxCmd represents the wav2adx command
var wav2adxCmd = &cobra.Command{
	Use:   "wav2adx",
	Short: "Converts WAV to ADX",
	Long:  `Converts WAV to ADX`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		adxtools.Wav2Adx(args[0])
	},
}

func init() {
	rootCmd.AddCommand(wav2adxCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wav2adxCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wav2adxCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
