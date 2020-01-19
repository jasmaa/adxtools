package adx

import (
	"encoding/binary"
	"os"
)

// Header contains ADX header data
type header struct {
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

// Read reads ADX file header
func (header *header) Read(fname string) {

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
	header.copyrightOffset = binary.BigEndian.Uint16(buffer[0x02:0x04])
	header.encodingType = buffer[0x04]
	header.blockSize = buffer[0x05]
	header.sampleBitdepth = buffer[0x06]
	header.channelCount = buffer[0x07]
	header.sampleRate = binary.BigEndian.Uint32(buffer[0x08:0x0C])
	header.totalSamples = binary.BigEndian.Uint32(buffer[0x0C:0x10])
	header.highpassFrequency = binary.BigEndian.Uint16(buffer[0x10:0x12])
	header.version = buffer[0x12]
	header.flags = buffer[0x13]

	// Version specific
	switch header.version {

	case 3:
		header.loopAlignmentSamples = binary.BigEndian.Uint16(buffer[0x14:0x16])
		header.loopEnabled = binary.BigEndian.Uint32(buffer[0x18:0x1C]) == 1
		header.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x1C:0x20])
		header.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x20:0x24])
		header.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x24:0x28])
		header.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
	case 4:
		header.loopEnabled = binary.BigEndian.Uint32(buffer[0x24:0x28]) == 1
		header.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
		header.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x2C:0x30])
		header.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x30:0x34])
		header.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x34:0x38])

	}
}
