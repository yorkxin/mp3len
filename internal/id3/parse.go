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

const id3v2Flag = "ID3" // first 3 bytes of an MP3 file with ID3v2 tag
const lenOfHeader = 10  // fixed length defined by ID3v2 spec

const textEncodingLatin1 = 0x00
const textEncodingUTF16 = 0x01

type Frame struct {
	ID     string // 4-char
	Size   uint32
	Flags  uint16
	Data   []byte
	Offset uint32 // byte offset from the id3 header (i.e. first frame offset = 0x0A)
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
func calculateID3TagSize(data []byte) uint32 {
	var size uint32 = 0

	// FIXME: handle len(data) < 4
	for place := 0; place < 4; place++ {
		value := data[place]
		size += uint32(value) << ((3 - place) * 7)
	}

	return size
}

func readNextFrame(r io.Reader) (frame *Frame, totalRead int, err error) {
	header := make([]byte, 10)
	n, err := r.Read(header)
	totalRead += n
	if err != nil {
		return
	}

	// Frame ID       $xx xx xx xx (four characters)
	// Size           $xx xx xx xx
	// Flags          $xx xx

	id := string(header[0:4])
	size := binary.BigEndian.Uint32(header[4:8])
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

// ReadFrames reads all ID3 tags
//
// returns total bytes of ID3 data (header + frames) and slice of frames
func ReadFrames(r io.Reader) (uint32, []Frame, error) {
	/*
		https://id3.org/id3v2.3.0
		ID3v2/file identifier   "ID3"
		ID3v2 version           $03 00
		ID3v2 flags             %abc00000
		ID3v2 size              4 * %0xxxxxxx
	*/
	frames := make([]Frame, 0)

	header := make([]byte, lenOfHeader)
	_, err := r.Read(header)

	if err != nil {
		return 0, nil, err
	}

	if string(header[0:3]) != id3v2Flag {
		err = fmt.Errorf("does not look like an MP3 file")
		return 0, nil, err
	}

	// ignoring [3] and [4] (version)
	// ignoring [5] (8-bit, flags)
	size := calculateID3TagSize(header[6:lenOfHeader]) // 6, 7, 8, 9

	var offset uint32 = 0
	for offset < size {
		frame, totalRead, err := readNextFrame(r)
		if err != nil {
			err = fmt.Errorf("read frame failed at %04X, err: %s", offset, err)
			break
		}

		frame.Offset = offset + lenOfHeader
		offset += uint32(totalRead)

		if frame.Size == 0 {
			// reached end of id3tags. Bye
			break
		} else {
			frames = append(frames, *frame)
		}
	}

	total := offset + lenOfHeader

	// read through all 0's between id3tags and mp3 audio frame
	// Example: Hindenburg Journalist, pads ID3 tags until 64KB.
	for discard := make([]byte, 1); offset < size; offset++ {
		_, err := r.Read(discard)
		if err != nil {
			return total, frames, err
		}
	}

	return total, frames, nil
}
