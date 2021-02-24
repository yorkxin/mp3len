package id3

import (
	"fmt"
	"io"
	"io/ioutil"
)

// Tag is the whole ID3 Tag block, including a Header and many FrameWithOffset.
type Tag struct {
	Header      Header
	Frames      []FrameWithOffset
	PaddingSize int
}

// Size returns total bytes of the ID3 tag, excluding header, including padding.
//
// Excerpt from spec:
//
//  https://id3.org/id3v2.3.0#ID3v2_header
//
//	The ID3v2 tag size is the size of the complete tag after unsychronisation,
//	including padding, excluding the header but not excluding the extended
//	header (total tag size - 10). Only 28 bits (representing up to 256MB) are
//	used in the size description to avoid the introducuction of
//	'false syncsignals'.
func (t *Tag) Size() int {
	return t.Header.Size
}

// TotalSize returns total bytes of the ID3 tag, including header, frames and padding.
func (t *Tag) TotalSize() int {
	return t.Header.Size + lenOfHeader
}

type FrameWithOffset struct {
	Frame

	// Offset is the location in the data part, excluding tag header (10 bytes)
	// To get the offset from the first byte of ID3 tag, plus 10.
	Offset int
}

// String returns Frame.String() with offset prefix.
func (f *FrameWithOffset) String() string {
	return fmt.Sprintf("[%04X] %s", f.Offset+lenOfHeader, f.Frame.String())
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

	headerBytes := [lenOfHeader]byte{}
	n, err := r.Read(headerBytes[:])
	totalRead += n

	if err != nil {
		return nil, totalRead, err
	}

	header, err := parseHeader(headerBytes)

	if err != nil {
		return nil, totalRead, err
	}

	frames := make([]FrameWithOffset, 0)

	// Avoid read exceeding ID3 Tag boundary
	lr := io.LimitReader(r, int64(header.Size))

	// offset from header
	offset := 0
	for offset < header.Size {
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

		frameWithOffset := FrameWithOffset{Frame: *frame, Offset: offset}
		frames = append(frames, frameWithOffset)
		offset += n
	}

	paddingSize := header.Size + lenOfHeader - totalRead

	// discard padding bytes
	nDiscarded, err := io.CopyN(ioutil.Discard, lr, int64(paddingSize))
	totalRead += int(nDiscarded)

	if err != nil {
		return nil, totalRead, err
	}

	return &Tag{
		Header:      *header,
		Frames:      frames,
		PaddingSize: paddingSize,
	}, totalRead, nil
}
