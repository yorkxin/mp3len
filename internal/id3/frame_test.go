package id3

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func isLatin1Compatible(data []byte) bool {
	for _, c := range data {
		if c < 0x20 || c > 0x7F {
			return false
		}
	}

	return true
}

func encodeUTF16String(data []byte) ([]byte, error) {
	utf16Encoding := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()
	utf16Reader := transform.NewReader(bytes.NewReader(data), utf16Encoding)
	utf16, err := ioutil.ReadAll(utf16Reader)

	if err != nil {
		return nil, err
	}

	return utf16, nil
}

// generateFrame is a test helper to generate ID3 frame payload.
//
// This function can be deprecated once id3 package supports tag encoding.
func generateFrame(id string, data []byte, flags [2]byte) []byte {
	// FIXME: support TXXX and WXXX that can specify encoding flag and description.
	var buf bytes.Buffer
	buf.WriteString(id)

	var actualData bytes.Buffer

	// Text Frame as defined by ID3 v2.3
	if id[0] == 'T' || id[0] == 'W' {
		// Check encoding. If Latin then write directly, otherwise write UTF-16.
		if isLatin1Compatible(data) {
			actualData.WriteByte(textEncodingLatin1)
			actualData.Write(data)
			actualData.WriteByte(0x00) // \0
		} else {
			actualData.WriteByte(textEncodingUTF16)
			str, err := encodeUTF16String(data)
			if err != nil {
				panic(err)
			}
			actualData.Write(str)
			actualData.Write([]byte{0x00, 0x00}) // \0\0
		}
	} else {
		actualData.Write(data)
	}

	err := binary.Write(&buf, binary.BigEndian, int32(actualData.Len())) // size
	if err != nil {
		panic(err)
	}

	buf.Write(flags[:])
	buf.Write(actualData.Bytes())
	return buf.Bytes()
}

func Test_readNextFrame(t *testing.T) {
	sampleTextFrame := generateFrame("TIT2", []byte("Foo Bar"), [2]byte{})

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name          string
		args          args
		wantFrame     *Frame
		wantTotalRead int
		wantErr       bool
	}{
		{
			name: "Text Frame",
			args: args{bytes.NewReader(sampleTextFrame)},
			wantFrame: &Frame{
				ID:    "TIT2",
				size:  9,
				Flags: 0,
				Data:  []byte("\x00Foo Bar\x00"),
			},
			wantTotalRead: 19,
			wantErr:       false,
		},
		{
			name: "Text Frame with non-Latin1 chars",
			args: args{bytes.NewReader(generateFrame("TIT2", []byte("世界你好"), [2]byte{}))},
			wantFrame: &Frame{
				ID:    "TIT2",
				size:  13,
				Flags: 0,
				Data:  []byte("\x01\xFE\xFF\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00"),
			},
			wantTotalRead: 23,
			wantErr:       false,
		},
		{
			name: "Data Frame",
			args: args{bytes.NewReader(generateFrame("PRIV", []byte{0xDE, 0xAD, 0xBE, 0xEF}, [2]byte{}))},
			wantFrame: &Frame{
				ID:    "PRIV",
				size:  4,
				Flags: 0,
				Data:  []byte("\xDE\xAD\xBE\xEF"),
			},
			wantTotalRead: 14,
			wantErr:       false,
		},
		{
			name:          "Padding",
			args:          args{bytes.NewReader(make([]byte, 20))},
			wantFrame:     nil,
			wantTotalRead: 10, // it reads 10 bytes of header, found all 0, then abort.
			wantErr:       false,
		},
		{
			name:          "Error: Failed to read header",
			args:          args{io.LimitReader(bytes.NewReader(sampleTextFrame), 5)},
			wantFrame:     nil,
			wantTotalRead: 5,
			wantErr:       true,
		},
		{
			name:          "Error: Failed to read frame data",
			args:          args{io.LimitReader(bytes.NewReader(sampleTextFrame), 11)},
			wantFrame:     nil,
			wantTotalRead: 11,
			wantErr:       true,
		},
		{
			name:          "Error: Invalid frame ID",
			args:          args{bytes.NewReader(generateFrame(string([]byte{0xde, 0xad, 0xbe, 0xef}), []byte("Foo Bar"), [2]byte{}))},
			wantFrame:     nil,
			wantTotalRead: 10,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrame, gotTotalRead, err := readNextFrame(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readNextFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFrame, tt.wantFrame) {
				t.Errorf("readNextFrame() gotFrame = %v, want %v", gotFrame, tt.wantFrame)
			}
			if gotTotalRead != tt.wantTotalRead {
				t.Errorf("readNextFrame() gotTotalRead = %v, want %v", gotTotalRead, tt.wantTotalRead)
			}
		})
	}
}
