package id3

import (
	"bytes"
	"errors"
	"fmt"
)

var id3v2Flag = []byte("ID3") // first 3 bytes of an MP3 file with ID3v2 tag
const lenOfHeader = 10        // fixed length defined by ID3v2 spec

type Header struct {
	Version  uint8
	Revision uint8
	Flags    uint8

	// Parsed from header payload by decodeTagSize.
	// The Size does not include header itself (always 10 bytes)
	Size int
}

func (h *Header) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.Write(id3v2Flag)
	buf.WriteByte(h.Version)
	buf.WriteByte(h.Revision)
	buf.WriteByte(h.Flags)
	buf.Write(encodeTagSize(h.Size))
	return buf.Bytes()
}

// String returns human-readable description of the ID3 header
func (h *Header) String() string {
	// TODO: describe flags
	return fmt.Sprintf("format=ID3v2.%d.%d, Size=%d", h.Version, h.Revision, h.Size)
}

// decodeTagSize returns an integer from 4-byte (32-bit) input.
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
func decodeTagSize(data []byte) int {
	size := 0

	// FIXME: handle len(data) < 4
	for place := 0; place < 4; place++ {
		value := data[place] & 0b01111111 // effect bits are lower 7 bits
		size += int(value) << ((3 - place) * 7)
	}

	return size
}

func encodeTagSize(size int) []byte {
	data := make([]byte, 4)

	for place := 3; place >= 0; place-- {
		data[place] = uint8(size & 0b01111111) // effect bits are lower 7 bits
		size >>= 7
	}

	return data
}

func parseHeader(headerBytes [lenOfHeader]byte) (*Header, error) {
	if bytes.Compare(headerBytes[0:3], id3v2Flag) != 0 {
		return nil, errors.New("invalid ID3 header")
	}

	version := headerBytes[3]
	revision := headerBytes[4]
	flags := headerBytes[5]
	size := decodeTagSize(headerBytes[6:lenOfHeader]) // 6, 7, 8, 9

	return &Header{
		Version:  version,
		Revision: revision,
		Flags:    flags,
		Size:     size,
	}, nil
}
