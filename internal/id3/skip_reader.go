package id3

import (
	"io"
	"io/ioutil"
)

// SkipReader reads through the whole ID3v2 tag block, but does not store
// anything in the memory. Useful for ignoring ID3v2 tag section.
type SkipReader struct {
	r io.Reader
	n int // n bytes that has been read
}

func NewSkipReader(r io.Reader) *SkipReader {
	return &SkipReader{r: r}
}

func (s *SkipReader) ReadThrough() (int, error) {
	header := new(tagHeader)
	n, err := readTagHeader(s.r, header)
	s.n += n

	if err != nil {
		return s.n, err
	}

	// Avoid read exceeding ID3 Tag boundary
	s.r = io.LimitReader(s.r, int64(header.size))

	nDiscarded, err := io.CopyN(ioutil.Discard, s.r, int64(header.size))
	s.n += int(nDiscarded)

	if err != nil {
		return s.n, err
	}

	return s.n, nil
}
