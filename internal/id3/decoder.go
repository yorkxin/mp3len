package id3

import (
	"bytes"
	"encoding/binary"
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

type Decoder struct {
	r    io.Reader
	n    int // n bytes that has already been read
	size int // total size of the tag payload, excluding header

	tag *Tag
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Decode() (*Tag, error) {
	header := make([]byte, 10)
	n, err := io.ReadFull(d.r, header)
	d.n += n

	if err != nil {
		return nil, err
	}

	if bytes.Compare(header[0:3], id3v2Flag) != 0 {
		return nil, errors.New("invalid ID3 header")
	}

	d.tag = new(Tag)
	d.tag.Version = header[3]
	d.tag.Revision = header[4]
	d.tag.Flags = header[5]

	d.size = decodeTagSize(header[6:]) // 6, 7, 8, 9

	d.tag.Frames = make([]Frame, 0)

	// Avoid read exceeding ID3 Tag boundary
	d.r = io.LimitReader(d.r, int64(d.size))

	// offset from header
	for {
		frame, err := d.readFrame()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read frame failed at %04X, err: %s", d.n, err)
		}

		if frame == nil {
			// reached padding. Bye
			break
		}

		d.tag.Frames = append(d.tag.Frames, *frame)
	}

	d.tag.PaddingSize = d.size + lenOfHeader - d.n

	// discard padding bytes
	nDiscarded, err := io.CopyN(ioutil.Discard, d.r, int64(d.tag.PaddingSize))
	d.n += int(nDiscarded)

	if err != nil {
		return nil, err
	}

	return d.tag, nil
}

// readFrame reads an ID3 frame from the reader.
//
// Returns a pointer to Frame and total bytes read (int) if successful.
//
// Returns nil *Frame and nil error when all data are 0x00 (padding). The caller
// should discard all the remaining data up to end of ID3 tag.
func (d *Decoder) readFrame() (*Frame, error) {
	header := [10]byte{}
	n, err := io.ReadFull(d.r, header[:])
	d.n += n
	if err != nil {
		return nil, err
	}

	allZero := [10]byte{}

	if bytes.Compare(header[:], allZero[:]) == 0 {
		// Reached padding. Exit.
		return nil, nil
	}

	// Frame ID       $xx xx xx xx (four characters)
	// Size           $xx xx xx xx
	// Flags          $xx xx

	// verify if the id is a valid string
	idRaw := header[0:4]
	for _, c := range idRaw {
		if !(('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')) {
			return nil, fmt.Errorf("invalid header: %v", idRaw)
		}
	}

	id := string(idRaw)

	// It's safe to represent size as a 32-bit signed int, even if the spec says
	// it uses 32-bit integer without specifying it's signed or unsigned,
	// because the Size section of tag header can only store an 28-bit signed
	// integer.
	//
	// See decodeTagSize for details.
	//
	// FIXME: find a way to read signed int directly, without explicit type conversion
	size := int(binary.BigEndian.Uint32(header[4:8]))
	flags := binary.BigEndian.Uint16(header[8:10])
	data := make([]byte, size)
	// In case of HTTP response body, r is a bufio.Reader, and in some cases
	// r.Read() may not fill the whole len(data). Using io.ReadFull ensures it
	// fills the whole len(data) slice.
	n, err = io.ReadFull(d.r, data)

	d.n += n

	if err != nil {
		return nil, err
	}

	frame := new(Frame)
	frame.ID = id
	frame.Flags = flags
	frame.Data = data

	return frame, nil
}

// InputOffset returns how many bytes that the decoder has read so far.
func (d *Decoder) InputOffset() int {
	return d.n
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
