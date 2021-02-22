package id3

import (
	"bytes"
	"fmt"
	"io"
)

var id3v2Flag = []byte("ID3") // first 3 bytes of an MP3 file with ID3v2 tag
const lenOfHeader = 10        // fixed length defined by ID3v2 spec

type Header struct {
	Version  int
	Revision int
	Flags    uint8

	// total size of the ID3 tags
	Size int
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
		value := data[place]
		size += int(value) << ((3 - place) * 7)
	}

	return size
}

// TODO: extract below to parse.go
type FrameWithOffset struct {
	Frame

	// location in the input stream
	Offset int
}

// String returns Frame.String() with offset prefix.
func (f *FrameWithOffset) String() string {
	return fmt.Sprintf("[%04X] %s", f.Offset, f.Frame.String())
}

// ReadFrames reads all ID3 tags
//
// returns total bytes of ID3 data (header + frames) and slice of frames
func ReadFrames(r io.Reader) (int, []FrameWithOffset, error) {
	/*
		https://id3.org/id3v2.3.0
		ID3v2/file identifier   "ID3"
		ID3v2 version           $03 00
		ID3v2 flags             %abc00000
		ID3v2 size              4 * %0xxxxxxx
	*/
	header := [lenOfHeader]byte{}

	frames := make([]FrameWithOffset, 0)

	_, err := r.Read(header[:])

	if err != nil {
		return 0, nil, err
	}

	if bytes.Compare(header[0:3], id3v2Flag) != 0 {
		err = fmt.Errorf("does not look like an MP3 file")
		return 0, nil, err
	}

	// ignoring [3] and [4] (version)
	// ignoring [5] (8-bit, flags)
	size := calculateID3TagSize(header[6:lenOfHeader]) // 6, 7, 8, 9

	offset := 0
	for offset < size {
		frame, totalRead, err := readNextFrame(r)
		if err != nil {
			err = fmt.Errorf("read frame failed at %04X, err: %s", offset, err)
			break
		}

		frameWithOffset := FrameWithOffset{Frame: *frame, Offset: offset + lenOfHeader}

		offset += totalRead

		if frame.Size() == 0 {
			// reached end of id3tags. Bye
			break
		} else {
			frames = append(frames, frameWithOffset)
		}
	}

	total := offset + lenOfHeader

	discard := [1]byte{}

	// read through all 0's between id3tags and mp3 audio frame
	// Example: Hindenburg Journalist, pads ID3 tags until 64KB.
	for ; offset < size; offset++ {
		_, err := r.Read(discard[:])
		if err != nil {
			return total, frames, err
		}
	}

	return total, frames, nil
}
