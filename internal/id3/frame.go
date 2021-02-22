package id3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const textEncodingLatin1 = 0x00
const textEncodingUTF16 = 0x01

type Frame struct {
	ID string // 4-char

	// It's safe to store Size as a signed int, even if the spec says it uses
	// 32-bit integer without specifying it's signed or unsigned, because
	// the size section of tag header can only store an 28-bit signed integer.
	//
	// See calculateID3TagSize for details.
	Size  int
	Flags uint16
	Data  []byte

	// Byte offset from the id3 header (First frame offset (0x0A) + X).
	//
	// Not decoded from frame data, but added by readNextFrame() function.
	//
	// It's safe to store Offset as a signed int, see comments for Size field.
	Offset int
}

func (frame *Frame) String() string {
	var content string

	if frame.hasText() {
		var err error
		content, err = decodeString(frame.Data)

		if err != nil {
			content = binaryView(frame.Data, 100)
		}
	} else {
		content = binaryView(frame.Data, 100)
	}

	return fmt.Sprintf("[%04X] %s %-5d %016b %s", frame.Offset, frame.ID, frame.Size, frame.Flags, content)
}

func (frame *Frame) hasText() bool {
	return frame.ID[0] == 'T' || frame.ID[0] == 'W'
}

// TextContent returns a string (UTF-8) decoded from frame data, if the data is
// a text frame or URL frame. For other kinds of frames, an empty string will
// be returned, and the secondary return value will be bool(false).
//
// FIXME: support TXXX and WXXX which has 3 sections, encoding flag, description
// and text, separated by 0x00{1,2}
func decodeString(data []byte) (string, error) {
	// First byte is encoding flag
	if data[0] == textEncodingLatin1 {
		return string(data[1:]), nil
	} else if data[0] == textEncodingUTF16 {
		return decodeUTF16String(data[1:])
	} else {
		// Undefined text encoding
		return "", fmt.Errorf("unable to decode string")
	}
}

func decodeUTF16String(buf []byte) (string, error) {
	utf16Encoding := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder()
	utf16Reader := transform.NewReader(bytes.NewReader(buf), utf16Encoding)
	utf8, err := ioutil.ReadAll(utf16Reader)

	if err != nil {
		return "", err
	}

	return string(utf8), nil
}

func binaryView(buf []byte, max int) string {
	// Fallback to binary view
	if len(buf) > max {
		return fmt.Sprintf("%x[...]", buf[:max])
	}

	return fmt.Sprintf("%x", buf)
}

func readNextFrame(r io.Reader) (frame *Frame, totalRead int, err error) {
	header := [10]byte{}
	n, err := r.Read(header[:])
	totalRead += n
	if err != nil {
		return
	}

	// Frame ID       $xx xx xx xx (four characters)
	// Size           $xx xx xx xx
	// Flags          $xx xx

	id := string(header[0:4])

	// ignoring sign bit. See comments for Size field.
	// FIXME: find a way to read signed int directly, without explicit type conversion
	size := int(binary.BigEndian.Uint32(header[4:8]))
	flags := binary.BigEndian.Uint16(header[8:10])
	data := make([]byte, size)
	// In case of HTTP response body, r is a bufio.Reader, and in some cases
	// r.Read() may not fill the whole len(data). Using io.ReadFull ensures it
	// fills the whole len(data) slice.
	// FIXME: handle err
	n, _ = io.ReadFull(r, data)
	totalRead += n

	frame = &Frame{
		ID:    id,
		Size:  size,
		Flags: flags,
		Data:  data,
	}

	return
}
