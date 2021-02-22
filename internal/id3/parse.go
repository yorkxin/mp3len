package id3

import (
	"fmt"
	"io"
)

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
	headerBytes := [lenOfHeader]byte{}
	_, err := r.Read(headerBytes[:])

	if err != nil {
		return 0, nil, err
	}

	header, err := parseHeader(headerBytes)

	if err != nil {
		return 0, nil, err
	}

	frames := make([]FrameWithOffset, 0)

	offset := 0
	for offset < header.Size() {
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
	for ; offset < header.Size(); offset++ {
		_, err := r.Read(discard[:])
		if err != nil {
			return total, frames, err
		}
	}

	return total, frames, nil
}
