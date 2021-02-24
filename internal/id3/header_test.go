package id3

import (
	"reflect"
	"testing"
)

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

func TestHeader_Bytes(t *testing.T) {
	type fields struct {
		Version  uint8
		Revision uint8
		Flags    uint8
		Size     int
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				Version:  2,
				Revision: 0,
				Flags:    0xFF,
				Size:     256,
			},
			want: []byte("ID3\x02\x00\xFF\x00\x00\x02\x00"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{
				Version:  tt.fields.Version,
				Revision: tt.fields.Revision,
				Flags:    tt.fields.Flags,
				Size:     tt.fields.Size,
			}
			if got := h.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
