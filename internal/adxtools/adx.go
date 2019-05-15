// ADX data

package adxtools

import (
	"encoding/binary"
	"os"
)

type ADX struct {
	copyrightOffset      uint16
	encodingType         uint8
	blockSize            uint8
	sampleBitdepth       uint8
	channelCount         uint8
	sampleRate           uint32
	totalSamples         uint32
	highpassFrequency    uint16
	version              uint8
	flags                uint8
	loopAlignmentSamples uint16
	loopEnabled          bool
	loopBeginSampleIndex uint32
	loopBeginByteIndex   uint32
	loopEndSampleIndex   uint32
	loopEndByteIndex     uint32
}

func (adx *ADX) ReadHeader(fname string) {

	buffer := make([]byte, 0x40)
	f, err := os.Open(fname)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	binary.Read(f, binary.LittleEndian, buffer)

	// Check magic
	if !(buffer[0x0] == 0x80 && buffer[0x1] == 0x00) {
		panic("Magic does not match")
	}

	// Write header metadata
	adx.copyrightOffset = binary.BigEndian.Uint16(buffer[0x02:0x04])
	adx.encodingType = buffer[0x04]
	adx.blockSize = buffer[0x05]
	adx.sampleBitdepth = buffer[0x06]
	adx.channelCount = buffer[0x07]
	adx.sampleRate = binary.BigEndian.Uint32(buffer[0x08:0x0C])
	adx.totalSamples = binary.BigEndian.Uint32(buffer[0x0C:0x10])
	adx.highpassFrequency = binary.BigEndian.Uint16(buffer[0x10:0x12])
	adx.version = buffer[0x12]
	adx.flags = buffer[0x13]

	// version specific
	if adx.version == 3 {
		adx.loopAlignmentSamples = binary.BigEndian.Uint16(buffer[0x14:0x16])
		adx.loopEnabled = binary.BigEndian.Uint32(buffer[0x18:0x1C]) == 1
		adx.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x1C:0x20])
		adx.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x20:0x24])
		adx.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x24:0x28])
		adx.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
	} else if adx.version == 4 {
		adx.loopEnabled = binary.BigEndian.Uint32(buffer[0x24:0x28]) == 1
		adx.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
		adx.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x2C:0x30])
		adx.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x30:0x34])
		adx.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x34:0x38])
	}
}
