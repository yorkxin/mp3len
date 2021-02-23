package id3

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"testing"
)

func generateFrame(id string, data []byte, flags [2]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(id)
	err := binary.Write(&buf, binary.BigEndian, int32(len(data))) // size
	if err != nil {
		panic(err)
	}

	buf.Write(flags[:])
	buf.Write(data)
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
				size:  len("Foo Bar"),
				Flags: 0,
				Data:  []byte("Foo Bar"),
			},
			wantTotalRead: len("Foo Bar") + 10,
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
