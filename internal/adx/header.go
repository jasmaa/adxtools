package adx

import (
	"encoding/binary"
	"os"
)

// Header contains ADX header data
type header struct {
	copyrightOffset      uint
	encodingType         uint
	blockSize            uint
	sampleBitdepth       uint
	channelCount         uint
	sampleRate           uint
	totalSamples         uint
	highpassFrequency    uint
	version              uint
	flags                uint
	loopAlignmentSamples uint
	loopEnabled          bool
	loopBeginSampleIndex uint
	loopBeginByteIndex   uint
	loopEndSampleIndex   uint
	loopEndByteIndex     uint
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
	header.copyrightOffset = uint(binary.BigEndian.Uint16(buffer[0x02:0x04]))
	header.encodingType = uint(buffer[0x04])
	header.blockSize = uint(buffer[0x05])
	header.sampleBitdepth = uint(buffer[0x06])
	header.channelCount = uint(buffer[0x07])
	header.sampleRate = uint(binary.BigEndian.Uint32(buffer[0x08:0x0C]))
	header.totalSamples = uint(binary.BigEndian.Uint32(buffer[0x0C:0x10]))
	header.highpassFrequency = uint(binary.BigEndian.Uint16(buffer[0x10:0x12]))
	header.version = uint(buffer[0x12])
	header.flags = uint(buffer[0x13])

	// Version specific
	switch header.version {

	case 3:
		header.loopAlignmentSamples = uint(binary.BigEndian.Uint16(buffer[0x14:0x16]))
		header.loopEnabled = binary.BigEndian.Uint32(buffer[0x18:0x1C]) == 1 // ignoring [0x16:0x18] for now
		header.loopBeginSampleIndex = uint(binary.BigEndian.Uint32(buffer[0x1C:0x20]))
		header.loopBeginByteIndex = uint(binary.BigEndian.Uint32(buffer[0x20:0x24]))
		header.loopEndSampleIndex = uint(binary.BigEndian.Uint32(buffer[0x24:0x28]))
		header.loopEndByteIndex = uint(binary.BigEndian.Uint32(buffer[0x28:0x2C]))

	case 4:
		header.loopEnabled = binary.BigEndian.Uint32(buffer[0x24:0x28]) == 1
		header.loopBeginSampleIndex = uint(binary.BigEndian.Uint32(buffer[0x28:0x2C]))
		header.loopBeginByteIndex = uint(binary.BigEndian.Uint32(buffer[0x2C:0x30]))
		header.loopEndSampleIndex = uint(binary.BigEndian.Uint32(buffer[0x30:0x34]))
		header.loopEndByteIndex = uint(binary.BigEndian.Uint32(buffer[0x34:0x38]))

	}
}

// Write writes ADX file header
func (header *header) Write(fname string) {

	buffer := make([]byte, 0x40)
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	// Magic
	buffer[0x0] = 0x80
	buffer[0x1] = 0x00

	// Write header metadata
	binary.BigEndian.PutUint16(buffer[0x02:0x04], uint16(header.copyrightOffset))
	buffer[0x04] = byte(header.encodingType)
	buffer[0x05] = byte(header.blockSize)
	buffer[0x06] = byte(header.sampleBitdepth)
	buffer[0x07] = byte(header.channelCount)
	binary.BigEndian.PutUint32(buffer[0x08:0x0C], uint32(header.sampleRate))
	binary.BigEndian.PutUint32(buffer[0x0C:0x10], uint32(header.totalSamples))
	binary.BigEndian.PutUint16(buffer[0x10:0x12], uint16(header.highpassFrequency))
	buffer[0x12] = byte(header.version)
	buffer[0x13] = byte(header.flags)

	// Version specific
	switch header.version {

	case 3:
		binary.BigEndian.PutUint16(buffer[0x14:0x16], uint16(header.loopAlignmentSamples))

		binary.BigEndian.PutUint16(buffer[0x16:0x18], uint16(0))
		if header.loopEnabled {
			binary.BigEndian.PutUint32(buffer[0x18:0x1C], uint32(1))
		} else {
			binary.BigEndian.PutUint32(buffer[0x18:0x1C], uint32(0))
		}

		binary.BigEndian.PutUint32(buffer[0x1C:0x20], uint32(header.loopBeginSampleIndex))
		binary.BigEndian.PutUint32(buffer[0x20:0x24], uint32(header.loopBeginByteIndex))
		binary.BigEndian.PutUint32(buffer[0x24:0x28], uint32(header.loopEndSampleIndex))
		binary.BigEndian.PutUint32(buffer[0x28:0x2C], uint32(header.loopEndByteIndex))

	case 4:

		if header.loopEnabled {
			binary.BigEndian.PutUint32(buffer[0x24:0x28], uint32(1))
		} else {
			binary.BigEndian.PutUint32(buffer[0x24:0x28], uint32(0))
		}

		binary.BigEndian.PutUint32(buffer[0x28:0x2C], uint32(header.loopBeginSampleIndex))
		binary.BigEndian.PutUint32(buffer[0x2C:0x30], uint32(header.loopBeginByteIndex))
		binary.BigEndian.PutUint32(buffer[0x30:0x34], uint32(header.loopEndSampleIndex))
		binary.BigEndian.PutUint32(buffer[0x34:0x38], uint32(header.loopEndByteIndex))
	}

	binary.Write(f, binary.LittleEndian, buffer)
}
