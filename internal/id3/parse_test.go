package id3

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"
)

func openTestData(path string, limit int64) func(t *testing.T) io.Reader {
	return func(t *testing.T) io.Reader {
		f, err := os.Open(path)

		if err != nil {
			t.Fatal("Failed:", err)
		}

		t.Cleanup(func() {
			f.Close()
		})

		if limit > 0 {
			return io.LimitReader(f, limit)
		}

		return f
	}
}

func testBytes(buf []byte) func(t *testing.T) io.Reader {
	return func(t *testing.T) io.Reader {
		return bytes.NewReader(buf)
	}
}

func TestParse(t *testing.T) {
	type args struct {
		opener func(t *testing.T) io.Reader
	}

	type matchedTag struct {
		version, revision, flags uint8
		paddingSize              int
		frameLength              int
	}

	tagToMatchedTag := func(tag *Tag) *matchedTag {
		if tag == nil {
			return nil
		}

		return &matchedTag{
			version:     tag.Version,
			revision:    tag.Revision,
			flags:       tag.Flags,
			paddingSize: tag.PaddingSize,
			frameLength: len(tag.Frames),
		}
	}

	tests := []struct {
		name          string
		args          args
		wantTag       *matchedTag
		wantTotalRead int
		wantErr       bool
	}{
		{
			name: "Compact File",
			args: args{openTestData("./testdata/id3_compact.bin", 0)},
			wantTag: &matchedTag{
				version: 3, revision: 0, flags: 0,
				paddingSize: 0,
				frameLength: 16,
			},
			wantTotalRead: 330175,
			wantErr:       false,
		},
		{
			name: "Compact File",
			args: args{openTestData("./testdata/id3_compact.bin", 0)},
			wantTag: &matchedTag{
				version: 3, revision: 0, flags: 0,
				paddingSize: 0,
				frameLength: 16,
			},
			wantTotalRead: 330175,
			wantErr:       false,
		},
		{
			name: "Padded File",
			args: args{openTestData("./testdata/id3_padded.bin", 0)},
			wantTag: &matchedTag{
				version: 3, revision: 0, flags: 0,
				paddingSize: 53269,
				frameLength: 17,
			},
			wantTotalRead: 65536,
			wantErr:       false,
		},
		{
			name:          "Corrupted file",
			args:          args{openTestData("/dev/random", 0)},
			wantTag:       nil,
			wantTotalRead: 10, // tried to read header
			wantErr:       true,
		},
		{
			name: "Empty ID3 Tag",
			args: args{testBytes([]byte("ID3\x03\x00\xE0\x00\x00\x00\x00"))},
			wantTag: &matchedTag{
				version:     3,
				revision:    0,
				flags:       0b11100000,
				paddingSize: 0,
				frameLength: 0,
			},
			wantTotalRead: 10,
			wantErr:       false,
		},
		{
			name:          "Invalid leading bits",
			args:          args{testBytes([]byte("IE6\x03\x00\x00\x7F\x7F\x7F\x7F"))},
			wantTag:       nil,
			wantTotalRead: 10,
			wantErr:       true,
		},
		{
			name:          "Empty file",
			args:          args{openTestData("/dev/null", 0)},
			wantTag:       nil,
			wantTotalRead: 0,
			wantErr:       true,
		},
		{
			name:          "Reached EOF when reading a frame",
			args:          args{openTestData("./testdata/id3_compact.bin", 1024)},
			wantTag:       nil,
			wantTotalRead: 1024,
			wantErr:       true,
		},
		{
			name:          "Reached EOF when reading padding section",
			args:          args{openTestData("./testdata/id3_padded.bin", 60000)},
			wantTag:       nil,
			wantTotalRead: 60000,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, gotTotalRead, err := Parse(tt.args.opener(t))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(tagToMatchedTag(gotTag), tt.wantTag) {
				t.Errorf("Parse() gotTag = %v, want %#v", gotTag, tt.wantTag)
			}

			if gotTotalRead != tt.wantTotalRead {
				t.Errorf("Parse() gotTotalRead = %v, want %v", gotTotalRead, tt.wantTotalRead)
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
