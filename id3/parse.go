package id3

import (
	"encoding/binary"
	"fmt"
	"io"
)

const id3v2Flag = "ID3" // first 3 bytes of an MP3 file with ID3v2 tag

type ID3Frame struct {
	ID     string // 4-char
	Size   uint32
	Flags  uint16
	Data   []byte
	Offset uint32
}

func (frame *ID3Frame) String() string {
	var content string
	// TODO: detect printable instead of guessing by len()

	if frame.Size > 100 {
		content = fmt.Sprintf("%x[...]", frame.Data[0:100])
	} else {
		// FIXME: decode UTF-16
		content = string(frame.Data)
	}

	return fmt.Sprintf("[%04X] %s %-5d %016b %s", frame.Offset, frame.ID, frame.Size, frame.Flags, content)
}

// calculateID3TagSize returns an integer from 4-byte (32-bit) input.
// Per ID3v2 spec, the MSB of each byte is always 0 and ignored.
//
// NOTE: If data is longer than 4 bytes, only the first 4 bytes will be processed.
//
// For example:
//
//     (0x) 00 00 02 01
//     => _0000000 _0000000 _0000010 _0000001
//     => 10_0000001
//     => 0x101
//     => 257 (dec)
//
func calculateID3TagSize(data []byte) uint32 {
	var size uint32 = 0

	// FIXME: handle len(data) < 4
	for place := 0; place < 4; place++ {
		value := data[place]
		size += uint32(value) << ((3 - place) * 7)
	}

	return size
}

func readNextFrame(r io.Reader) (frame *ID3Frame, totalRead int, err error) {
	header := make([]byte, 10)
	n, err := r.Read(header)
	totalRead += n
	if err != nil {
		return
	}

	// Frame ID       $xx xx xx xx (four characters)
	// Size           $xx xx xx xx
	// Flags          $xx xx

	id := string(header[0:4])
	size := binary.BigEndian.Uint32(header[4:8])
	flags := binary.BigEndian.Uint16(header[8:10])
	data := make([]byte, size)
	// In case of HTTP response body, r is a bufio.Reader, and in some cases
	// r.Read() may not fill the whole len(data). Using io.ReadFull ensures it
	// fills the whole len(data) slice.
	// FIXME: handle err
	n, _ = io.ReadFull(r, data)
	totalRead += n

	frame = &ID3Frame{
		ID:    id,
		Size:  size,
		Flags: flags,
		Data:  data,
	}

	return
}

// ReadFrames reads all ID3 tags
func ReadFrames(r io.Reader) (size uint32, frames []ID3Frame, err error) {
	/*
		https://id3.org/id3v2.3.0
		ID3v2/file identifier   "ID3"
		ID3v2 version           $03 00
		ID3v2 flags             %abc00000
		ID3v2 size              4 * %0xxxxxxx
	*/
	frames = make([]ID3Frame, 0)

	header := make([]byte, 10)
	_, err = r.Read(header)

	if err != nil {
		return
	}

	if string(header[0:3]) != id3v2Flag {
		err = fmt.Errorf("does not look like an MP3 file")
		return
	}

	// ignoring [3] and [4] (version)
	// ignoring [5] (8-bit, flags)
	size = calculateID3TagSize(header[6:10]) // 6, 7, 8, 9

	var pos uint32 = uint32(len(header))
	for pos < size {
		frame, totalRead, err := readNextFrame(r)
		if err != nil {
			err = fmt.Errorf("read frame failed at %04X, err: %s", pos, err)
			break
		}

		frame.Offset = pos

		if frame.Size == 0 {
			// reached end of id3tags. Bye
			break
		} else {
			frames = append(frames, *frame)
		}

		pos += uint32(totalRead)
	}

	// read through all 0's between id3tags and mp3 audio frame
	remaining := size - pos
	discard := make([]byte, 1)
	for ; remaining > 0; remaining-- {
		r.Read(discard)
		pos++
	}

	return
}
