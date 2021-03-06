package id3

import (
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestNewSkipReader(t *testing.T) {
	anEmptyReader := emptyReader{}

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name string
		args args
		want *SkipReader
	}{
		{
			name: "OK",
			args: args{r: anEmptyReader},
			want: &SkipReader{r: anEmptyReader},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSkipReader(tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSkipReader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkipReader_ReadThrough(t *testing.T) {
	tests := []struct {
		filePath string
		want     int
		wantErr  bool
	}{
		{
			filePath: "./testdata/id3_compact.bin",
			want:     330175,
			wantErr:  false,
		},
		{
			filePath: "./testdata/id3_padded.bin",
			want:     65536,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("file: %s", tt.filePath), func(t *testing.T) {
			f := openTestData(tt.filePath, t)
			s := &SkipReader{r: f}
			got, err := s.ReadThrough()
			if (err != nil) != tt.wantErr {
				t.Errorf("SkipReader.ReadThrough() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SkipReader.ReadThrough() = %v, want %v", got, tt.want)
			}
		})
	}
}
