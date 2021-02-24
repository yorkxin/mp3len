package id3

import (
	"reflect"
	"testing"
)

func Test_calculateID3TagSize(t *testing.T) {
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
			if got := calculateID3TagSize(tt.args.data); got != tt.want {
				t.Errorf("calculateID3TagSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseHeader(t *testing.T) {
	type args struct {
		headerBytes [lenOfHeader]byte
	}
	tests := []struct {
		name    string
		args    args
		want    *Header
		wantErr bool
	}{
		{
			name: "Example",
			args: args{headerBytes: [10]byte{'I', 'D', '3', 0x03, 0x00, 0b11100000, 0x00, 0x00, 0x02, 0x01}},
			want: &Header{
				Version:  3,
				Revision: 0,
				Flags:    0b11100000,
				Size:     257,
			}, wantErr: false,
		},
		{
			name: "Huge Size",
			args: args{headerBytes: [10]byte{'I', 'D', '3', 0x03, 0x00, 0, 0x7F, 0x7F, 0x7F, 0x7F}},
			want: &Header{
				Version:  3,
				Revision: 0,
				Flags:    0,
				Size:     268435455,
			}, wantErr: false,
		},
		{
			name:    "Invalid leading bits",
			args:    args{headerBytes: [10]byte{'I', 'E', '6', 0x03, 0x00, 0, 0x7F, 0x7F, 0x7F, 0x7F}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHeader(tt.args.headerBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseHeader() got = %v, want %v", got, tt.want)
			}
		})
	}
}
