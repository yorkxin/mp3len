package id3

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func openTestData(path string, t *testing.T) io.Reader {
	f, err := os.Open(path)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		f.Close()
	})

	return f
}

func TestDecoder_Decode(t *testing.T) {
	type fields struct {
		r io.Reader
		n int
	}
	tests := []struct {
		name    string
		fields  fields
		wantTag *Tag
		wantErr bool
	}{
		{
			name:   "Empty Tag",
			fields: fields{r: bytes.NewReader([]byte("ID3\x03\x00\xE0\x00\x00\x00\x00"))},
			wantTag: &Tag{
				Version:     3,
				Revision:    0,
				Flags:       0b11100000,
				Frames:      []Frame{},
				PaddingSize: 0,
			},
			wantErr: false,
		},
		{
			name:    "Corrupted file",
			fields:  fields{r: bytes.NewReader([]byte("IE6\x03\x00\xE0\x00\x00\x00\x00"))},
			wantTag: nil,
			wantErr: true,
		},
		{
			name:    "Reached EOF when reading a frame",
			fields:  fields{r: io.LimitReader(openTestData("./testdata/id3_compact.bin", t), 1024)},
			wantTag: nil,
			wantErr: true,
		},
		{
			name:    "Reached EOF when reading header",
			fields:  fields{r: io.LimitReader(openTestData("./testdata/id3_compact.bin", t), 8)},
			wantTag: nil,
			wantErr: true,
		},
		{
			name:    "Reached EOF when reading padding section",
			fields:  fields{r: io.LimitReader(openTestData("./testdata/id3_padded.bin", t), 60000)},
			wantTag: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Decoder{
				r: tt.fields.r,
				n: tt.fields.n,
			}

			if rc, ok := tt.fields.r.(io.ReadCloser); ok {
				t.Cleanup(func() {
					rc.Close()
				})
			}
			tag, err := d.Decode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tag, tt.wantTag) {
				t.Errorf("Decode() tag = %v, want %v", tag, tt.wantTag)
			}
		})
	}
}

func TestDecoder_ActualFile(t *testing.T) {
	tests := []struct {
		filePath        string
		wantTagVersion  uint8
		wantTagRevision uint8
		wantTagFlags    uint8
		wantFrameLength int
		wantPaddingSize int
	}{
		{
			filePath:        "./testdata/id3_compact.bin",
			wantTagVersion:  3,
			wantTagRevision: 0,
			wantTagFlags:    0,
			wantFrameLength: 16,
			wantPaddingSize: 0,
		},
		{
			filePath:        "./testdata/id3_padded.bin",
			wantTagVersion:  3,
			wantTagRevision: 0,
			wantTagFlags:    0,
			wantFrameLength: 17,
			wantPaddingSize: 53269,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("file: %s", tt.filePath), func(t *testing.T) {
			f := openTestData(tt.filePath, t)

			decoder := new(Decoder)
			decoder.r = f

			tag, err := decoder.Decode()

			if err != nil {
				t.Errorf("Decode() error = %v", err)
			}

			if tag.Version != tt.wantTagVersion {
				t.Errorf("Decode() tag.Version = %v, want %v", tag.Version, tt.wantTagVersion)
			}

			if tag.Revision != tt.wantTagRevision {
				t.Errorf("Decode() tag.Revision = %v, want %v", tag.Revision, tt.wantTagRevision)
			}

			if tag.Flags != tt.wantTagFlags {
				t.Errorf("Decode() tag.Flags = %08b, want %08b", tag.Flags, tt.wantTagFlags)
			}

			if len(tag.Frames) != tt.wantFrameLength {
				t.Errorf("Decode() len(tag.Frames) = %v, want %v", len(tag.Frames), tt.wantFrameLength)
			}

			if tag.PaddingSize != tt.wantPaddingSize {
				t.Errorf("Decode() tag.PaddingSize = %v, want %v", tag.PaddingSize, tt.wantPaddingSize)
			}
		})
	}
}

func TestDecoder_readFrame(t *testing.T) {
	sampleTextFrame := generateTextFrame("TIT2", "Foo Bar", 0x0)

	type fields struct {
		r io.Reader
		n int
	}
	tests := []struct {
		name      string
		fields    fields
		wantFrame *Frame
		wantErr   bool
	}{
		{
			name:   "Text Frame",
			fields: fields{r: bytes.NewReader(sampleTextFrame)},
			wantFrame: &Frame{
				ID:    "TIT2",
				Flags: 0,
				Data:  []byte("\x00Foo Bar\x00"),
			},
			wantErr: false,
		},
		{
			name:   "Text Frame with non-Latin1 chars",
			fields: fields{r: bytes.NewReader(generateTextFrame("TIT2", "世界你好", 0x00))},
			wantFrame: &Frame{
				ID:    "TIT2",
				Flags: 0,
				Data:  []byte("\x01\xFE\xFF\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00"),
			},
			wantErr: false,
		},
		{
			name:   "Data Frame",
			fields: fields{r: bytes.NewReader(generateDataFrame("PRIV", []byte{0xDE, 0xAD, 0xBE, 0xEF}, 0x00))},
			wantFrame: &Frame{
				ID:    "PRIV",
				Flags: 0,
				Data:  []byte("\xDE\xAD\xBE\xEF"),
			},
			wantErr: false,
		},
		{
			name:      "Padding",
			fields:    fields{r: bytes.NewReader(make([]byte, 20))},
			wantFrame: nil,
			wantErr:   false,
		},
		{
			name:      "Error: Failed to read header",
			fields:    fields{r: io.LimitReader(bytes.NewReader(sampleTextFrame), 5)},
			wantFrame: nil,
			wantErr:   true,
		},
		{
			name:      "Error: Failed to read frame data",
			fields:    fields{r: io.LimitReader(bytes.NewReader(sampleTextFrame), 11)},
			wantFrame: nil,
			wantErr:   true,
		},
		{
			name:      "Error: Invalid frame ID",
			fields:    fields{r: bytes.NewReader(generateDataFrame(string([]byte{0xde, 0xad, 0xbe, 0xef}), []byte{}, 0x00))},
			wantFrame: nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Decoder{
				r: tt.fields.r,
				n: tt.fields.n,
			}

			frame, err := d.readFrame()

			if (err != nil) != tt.wantErr {
				t.Errorf("readFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(frame, tt.wantFrame) {
				t.Errorf("readFrame() frame = %v, want %v", frame, tt.wantFrame)
			}
		})
	}
}
func Test_decodeTagSize(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"spec", args{data: []byte{0x00, 0x00, 0x02, 0x01}}, 257},
		{"sample 1", args{data: []byte{0x00, 0x03, 0x7F, 0x76}}, 65526},
		{"max value", args{data: []byte{0x7F, 0x7F, 0x7F, 0x7F}}, 268435455},
		{"all set", args{data: []byte{0xFF, 0xFF, 0xFF, 0xFF}}, 268435455},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeTagSize(tt.args.data); got != tt.want {
				t.Errorf("decodeTagSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_encodeTagSize(t *testing.T) {
	type args struct {
		size int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"spec", args{257}, []byte{0x00, 0x00, 0x02, 0x01}},
		{"sample 1", args{65526}, []byte{0x00, 0x03, 0x7F, 0x76}},
		{"max value", args{268435455}, []byte{0x7F, 0x7F, 0x7F, 0x7F}},
		{"overflow", args{1<<32 - 1}, []byte{0x7F, 0x7F, 0x7F, 0x7F}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeTagSize(tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeTagSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

type emptyReader struct {
	eof bool
}

func (r emptyReader) Read(_ []byte) (int, error) {
	if r.eof {
		return 0, io.EOF
	}

	r.eof = true
	return 1, nil
}

func TestNewDecoder(t *testing.T) {
	anEmptyReader := emptyReader{}

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name string
		args args
		want *Decoder
	}{
		{name: "OK", args: args{r: anEmptyReader}, want: &Decoder{r: anEmptyReader, n: 0, size: 0, tag: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDecoder(tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDecoder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecoder_InputOffset(t *testing.T) {
	anEmptyReader := emptyReader{}

	type fields struct {
		r    io.Reader
		n    int
		size int
		tag  *Tag
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{name: "Returns current r.n", fields: fields{r: anEmptyReader, n: 999, size: 100, tag: nil}, want: 999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Decoder{
				r:    tt.fields.r,
				n:    tt.fields.n,
				size: tt.fields.size,
				tag:  tt.fields.tag,
			}
			if got := d.InputOffset(); got != tt.want {
				t.Errorf("Decoder.InputOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}
