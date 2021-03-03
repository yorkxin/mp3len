package id3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

const textEncodingLatin1 = 0x00
const textEncodingUTF16 = 0x01

// Frame holds data structure for an ID3v2 frame.
type Frame struct {
	ID    string // 4-char
	Flags uint16
	Data  []byte
}

// Text returns a string (UTF-8) decoded from frame data, if the data is
// a text frame or URL frame. For other kinds of frames, an empty string will
// be returned, and the secondary return value will be bool(false).
//
// FIXME: support TXXX and WXXX which has 3 sections, encoding flag, description
//        and text, separated by 0x00{1,2}
func (frame *Frame) Text() (string, error) {
	if !frame.hasText() {
		return "", fmt.Errorf("GetText(): Frame %q does not accept text content", frame.ID)
	}

	// First byte is encoding flag
	switch frame.Data[0] {
	case textEncodingLatin1:
		return decodeLatin1Text(frame.Data[1:]), nil
	case textEncodingUTF16:
		return decodeUTF16String(frame.Data[1:])
	default:
		// Undefined text encoding
		return "", fmt.Errorf("unable to decode string")
	}
}

// SetText sets the frame Data as the str. The existing Data will be overriden.
//
// str will be encoded in UTF16 if any rune is not Latin1. Returns error when
// encoding failed.
//
// Returns error when the frame definition does not accept text.
//
func (frame *Frame) SetText(str string) error {
	if !frame.hasText() {
		return fmt.Errorf("SetText(): Frame %q does not accept text content", frame.ID)
	}

	var buf bytes.Buffer

	// Check encoding. If Latin then write directly, otherwise write UTF-16.
	if isLatin1Compatible(str) {
		buf.WriteByte(textEncodingLatin1)
		buf.WriteString(str)
		buf.WriteByte(0x00)
	} else {
		buf.WriteByte(textEncodingUTF16)
		utf16Data, err := encodeUTF16String(str)
		if err != nil {
			return fmt.Errorf("SetText(): Encoding as UTF16 failed: %v", err)
		}
		buf.Write(utf16Data)
		buf.WriteByte(0x00)
		buf.WriteByte(0x00)
	}

	frame.Data = buf.Bytes()

	return nil
}

// Bytes returns the encoded bytes of the frame.
func (frame *Frame) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(frame.ID)

	err := binary.Write(&buf, binary.BigEndian, int32(len(frame.Data))) // size

	if err != nil {
		return nil, err
	}

	err = binary.Write(&buf, binary.BigEndian, frame.Flags)

	if err != nil {
		return nil, err
	}

	buf.Write(frame.Data)
	return buf.Bytes(), nil
}

// ByteSize calculates the bytes required to write the payload. It should be
// len(Data) + 10 bytes of header
func (frame *Frame) ByteSize() int {
	return len(frame.Data) + 10
}

func (frame *Frame) String() string {
	content, err := frame.Text()

	if err != nil {
		content = binaryView(frame.Data, 100)
	}

	return fmt.Sprintf("%s %-5d %016b %s", frame.ID, len(frame.Data), frame.Flags, content)
}

func (frame *Frame) hasText() bool {
	return frame.ID[0] == 'T' || frame.ID[0] == 'W'
}

func decodeLatin1Text(data []byte) string {
	terminus := len(data)

	for i, c := range data {
		if c == 0x0 {
			terminus = i
			break
		}
	}

	return string(data[:terminus])
}

func decodeUTF16String(buf []byte) (string, error) {
	if len(buf) < 2 {
		return "", errors.New("invalid UTF-16 payload")
	}

	reader := bytes.NewReader(buf)
	buf16Bit := make([]uint16, len(buf)/2)
	var err error

	if buf[0] == 0xFE && buf[1] == 0xFF {
		err = binary.Read(reader, binary.BigEndian, buf16Bit)
	} else if buf[0] == 0xFF && buf[1] == 0xFE {
		err = binary.Read(reader, binary.LittleEndian, buf16Bit)
	} else {
		err = errors.New("invalid UTF-16 payload (missing BOM)")
	}

	if err != nil {
		return "", err
	}

	terminus := len(buf16Bit)
	for i, r := range buf16Bit {
		if r == 0x0000 {
			terminus = i
			break
		}
	}

	utf8 := utf16.Decode(buf16Bit[:terminus])

	// ignore leading BOM
	return string(utf8[1:]), nil
}

func encodeUTF16String(str string) ([]byte, error) {
	utf16Data := utf16.Encode([]rune(str))
	buf := new(bytes.Buffer)
	buf.Write([]byte{0xFE, 0xFF}) // big endian
	err := binary.Write(buf, binary.BigEndian, utf16Data)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func binaryView(buf []byte, max int) string {
	// Fallback to binary view
	if len(buf) > max {
		return fmt.Sprintf("%x[...]", buf[:max])
	}

	return fmt.Sprintf("%x", buf)
}

func isLatin1Compatible(str string) bool {
	for _, r := range str {
		if r < 0x20 || r > 0x7F {
			return false
		}
	}

	return true
}
