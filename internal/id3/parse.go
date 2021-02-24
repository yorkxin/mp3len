package id3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

var id3v2Flag = []byte("ID3") // first 3 bytes of an MP3 file with ID3v2 tag
const lenOfHeader = 10        // fixed length defined by ID3v2 spec

// Tag is the whole ID3 Tag block, including a Header and many FrameWithOffset.
type Tag struct {
	// Header part
	Version  uint8
	Revision uint8
	Flags    uint8

	Frames      []Frame
	PaddingSize int
}

// Parse reads the whole ID3 tag from input r.
//
// Returns parsed Tag and totalRead bytes if successful.
//
func Parse(r io.Reader) (*Tag, int, error) {
	/*
		https://id3.org/id3v2.3.0
		ID3v2/file identifier   "ID3"
		ID3v2 version           $03 00
		ID3v2 flags             %abc00000
		ID3v2 size              4 * %0xxxxxxx
	*/
	totalRead := 0

	headerBytes := make([]byte, 10)
	n, err := io.ReadFull(r, headerBytes)
	totalRead += n

	if err != nil {
		return nil, totalRead, err
	}

	if bytes.Compare(headerBytes[0:3], id3v2Flag) != 0 {
		return nil, totalRead, errors.New("invalid ID3 header")
	}

	version := headerBytes[3]
	revision := headerBytes[4]
	flags := headerBytes[5]
	size := decodeTagSize(headerBytes[6:]) // 6, 7, 8, 9

	frames := make([]Frame, 0)

	// Avoid read exceeding ID3 Tag boundary
	lr := io.LimitReader(r, int64(size))

	// offset from header
	offset := 0
	for offset < size {
		frame, n, readErr := readNextFrame(lr)
		totalRead += n

		if readErr != nil {
			// Aborts reading further ID3 frames.
			err = fmt.Errorf("read frame failed at %04X, err: %s", offset+lenOfHeader, readErr)
			return nil, totalRead, err
		}

		if frame == nil {
			// reached padding. Bye
			break
		}

		frames = append(frames, *frame)
		offset += n
	}

	paddingSize := size + lenOfHeader - totalRead

	// discard padding bytes
	nDiscarded, err := io.CopyN(ioutil.Discard, lr, int64(paddingSize))
	totalRead += int(nDiscarded)

	if err != nil {
		return nil, totalRead, err
	}

	return &Tag{
		Version:     version,
		Revision:    revision,
		Flags:       flags,
		Frames:      frames,
		PaddingSize: paddingSize,
	}, totalRead, nil
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
