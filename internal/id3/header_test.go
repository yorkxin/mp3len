package id3

import "testing"

func Test_calculateID3TagSize(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{"spec", args{data: []byte{0x00, 0x00, 0x02, 0x01}}, 257},
		{"sample 1", args{data: []byte{0x00, 0x03, 0x7F, 0x76}}, 65526},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateID3TagSize(tt.args.data); got != tt.want {
				t.Errorf("calculateID3TagSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
