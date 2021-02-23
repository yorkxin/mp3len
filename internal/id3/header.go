package id3

import (
	"bytes"
	"errors"
)

var id3v2Flag = []byte("ID3") // first 3 bytes of an MP3 file with ID3v2 tag
const lenOfHeader = 10        // fixed length defined by ID3v2 spec

type Header struct {
	Version  int
	Revision int
	Flags    uint8

	// Parsed from header payload by calculateID3TagSize.
	// The size does not include header itself (always 10 bytes)
	size int
}

// calculateID3TagSize returns an integer from 4-byte (32-bit) input.
// Per ID3v2 spec, the MSB of each byte is always 0 and ignored.
//
// NOTE: If data is longer than 4 bytes, only the first 4 bytes will be processed.
//
// For example:
//
//     (0x) 00 00 02 01
//     => _0000000 _0000000 _0000010 _0000001
//     => 10_0000001
//     => 0x101
//     => 257 (dec)
//
func calculateID3TagSize(data []byte) int {
	size := 0

	// FIXME: handle len(data) < 4
	for place := 0; place < 4; place++ {
		value := data[place] & 0b01111111 // effect bits are lower 7 bits
		size += int(value) << ((3 - place) * 7)
	}

	return size
}

func parseHeader(headerBytes [lenOfHeader]byte) (*Header, error) {
	if bytes.Compare(headerBytes[0:3], id3v2Flag) != 0 {
		return nil, errors.New("invalid ID3 header")
	}

	version := int(headerBytes[3])
	revision := int(headerBytes[4])
	flags := headerBytes[5]
	size := calculateID3TagSize(headerBytes[6:lenOfHeader]) // 6, 7, 8, 9

	return &Header{
		Version:  version,
		Revision: revision,
		Flags:    flags,
		size:     size,
	}, nil
}
