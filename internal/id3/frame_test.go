package id3

import (
	"reflect"
	"testing"
)

func generateDataFrame(id string, data []byte, flags uint16) []byte {
	// FIXME: support TXXX and WXXX that can specify encoding flag and description.
	frame := Frame{ID: id, Data: data, Flags: flags}

	b, err := frame.Bytes()

	if err != nil {
		panic(err)
	}

	return b
}

func generateTextFrame(id string, str string, flags uint16) []byte {
	frame := Frame{ID: id, Flags: flags}

	err := frame.SetText(str)
	if err != nil {
		panic(err)
	}

	b, err := frame.Bytes()

	if err != nil {
		panic(err)
	}

	return b
}

func TestFrame_Text(t *testing.T) {
	type fields struct {
		ID    string
		Flags uint16
		Data  []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "Latin-1 Text",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x00My Fancy Album\x00"),
			},
			want:    "My Fancy Album",
			wantErr: false,
		}, {
			name: "Latin-1 Text, not terminated",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x00My Fancy Album"),
			},
			want:    "My Fancy Album",
			wantErr: false,
		},
		{
			name: "Latin-1 Text with data after string termination",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x00My Fancy Album\x00OLD TITLE"),
			},
			want:    "My Fancy Album",
			wantErr: false,
		},
		{
			name: "UTF-16 Text (Little Endian)",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01\xFE\xFF\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00"),
			},
			want:    "世界你好",
			wantErr: false,
		},
		{
			name: "UTF-16 Text (Big Endian)",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01\xFF\xFE\x16\x4E\x4C\x75\x60\x4F\x7D\x59\x00\x00"),
			},
			want:    "世界你好",
			wantErr: false,
		},
		{
			name: "UTF-16 Text, not terminated",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01\xFF\xFE\x16\x4E\x4C\x75\x60\x4F\x7D\x59"),
			},
			want:    "世界你好",
			wantErr: false,
		},
		{
			name: "UTF-16 Text with data after string termination",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01\xFE\xFF\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00\xAB\xCD\xEF"),
			},
			want:    "世界你好",
			wantErr: false,
		},
		{
			name: "Error: Invalid UTF-16 payload (Missing BOM)",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00"),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Error: Invalid UTF-16 payload (empty data)",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x01"),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Error: Invalid encoding flag",
			fields: fields{
				ID:    "TALB",
				Flags: 0,
				Data:  []byte("\x02"),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Error: Not a text frame",
			fields: fields{
				ID:    "PRIV",
				Flags: 0,
				Data:  []byte("\xDE\xAD\xBE\xEF"),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{
				ID:    tt.fields.ID,
				Flags: tt.fields.Flags,
				Data:  tt.fields.Data,
			}
			got, err := frame.Text()
			if (err != nil) != tt.wantErr {
				t.Errorf("Text() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Text() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrame_SetText(t *testing.T) {
	type fields struct {
		ID    string
		Flags uint16
		Data  []byte
	}
	type args struct {
		str string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData []byte
		wantErr  bool
	}{
		{
			name:     "Latin1 Text",
			fields:   fields{"TALB", 0, nil},
			args:     args{str: "My Fancy Album"},
			wantData: []byte("\x00My Fancy Album\x00"),
			wantErr:  false,
		},
		{
			name:     "Non-Latin1 Text",
			fields:   fields{"TALB", 0, nil},
			args:     args{str: "世界你好"},
			wantData: []byte("\x01\xFE\xFF\x4E\x16\x75\x4C\x4F\x60\x59\x7D\x00\x00"),
			wantErr:  false,
		},
		{
			name:     "Error: Non-Text Frame",
			fields:   fields{"PRIV", 0, nil},
			args:     args{str: "Test"},
			wantData: nil,
			wantErr:  true,
		},
		//	TODO: test a string that cannot be encoded as UTF-16
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{
				ID:    tt.fields.ID,
				Flags: tt.fields.Flags,
				Data:  tt.fields.Data,
			}
			if err := frame.SetText(tt.args.str); (err != nil) != tt.wantErr {
				t.Errorf("SetText() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.wantData, frame.Data) {
				t.Errorf("SetText() Data = %v, want %v", frame.Data, tt.wantData)
			}
		})
	}
}

func TestFrame_Bytes(t *testing.T) {
	type fields struct {
		ID    string
		Flags uint16
		Data  []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				ID:    "PRIV",
				Flags: 0x007F,
				Data:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
			want:    []byte("PRIV\x00\x00\x00\x04\x00\x7F\xDE\xAD\xBE\xEF"),
			wantErr: false,
		},
		{
			name: "Zero Length",
			fields: fields{
				ID:    "PRIV",
				Flags: 0x0000,
				Data:  []byte{},
			},
			want:    []byte("PRIV\x00\x00\x00\x00\x00\x00"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{
				ID:    tt.fields.ID,
				Flags: tt.fields.Flags,
				Data:  tt.fields.Data,
			}
			got, err := frame.Bytes()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrame_ByteSize(t *testing.T) {
	type fields struct {
		ID    string
		Flags uint16
		Data  []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "OK",
			fields: fields{
				ID:    "PRIV",
				Flags: 0x007F,
				Data:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
			want: 14,
		},
		{
			name: "Empty",
			fields: fields{
				ID:    "PRIV",
				Flags: 0x007F,
				Data:  nil,
			},
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{
				ID:    tt.fields.ID,
				Flags: tt.fields.Flags,
				Data:  tt.fields.Data,
			}
			if got := frame.ByteSize(); got != tt.want {
				t.Errorf("ByteSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
