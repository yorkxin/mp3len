package id3

import (
	"fmt"
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

func (t *Tag) String() string {
	return fmt.Sprintf("%s, frames=%d, paddingSize=%d", t.Header.String(), len(t.Frames), t.PaddingSize)
}

func TestParse(t *testing.T) {
	type args struct {
		opener func(t *testing.T) io.Reader
	}

	type matchedTag struct {
		header      Header
		paddingSize int
		frameLength int
	}

	tagToMatchedTag := func(tag *Tag) *matchedTag {
		if tag == nil {
			return nil
		}

		return &matchedTag{
			header:      tag.Header,
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
				header:      Header{Version: 3, Revision: 0, Flags: 0, size: 330165},
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
				header:      Header{Version: 3, Revision: 0, Flags: 0, size: 65526},
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
