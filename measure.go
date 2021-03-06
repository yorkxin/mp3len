package mp3len

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"mp3len/internal/id3"
	"mp3len/internal/mp3header"
)

// Metadata holds the parsed metadata of an MP3 input
type Metadata struct {
	duration  time.Duration // Estimated duration of the MP3
	tagSize   int
	mp3Header mp3header.MP3Header // MP3 Audio Frame Header (first frame only)
}

func (metadata *Metadata) calculateDuration(totalSize int64) {
	// Algorithm from https://www.factorialcomplexity.com/blog/how-to-get-a-duration-of-a-remote-mp3-file
	metadata.duration = time.Duration((totalSize - int64(metadata.tagSize+10)) / (int64(metadata.mp3Header.BitRate) / 8) * 1000000)

	if metadata.mp3Header.ChannelMode == mp3header.ChannelModeMono {
		metadata.duration *= 2
	}
}

func (metadata *Metadata) String(verbose bool) string {
	if !verbose {
		return metadata.duration.String()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintln("Duration:", metadata.duration.String()))

	sb.WriteString("Audio: ")
	sb.WriteString(metadata.mp3Header.String())
	sb.WriteByte('\n')

	sb.WriteString(fmt.Sprintf("ID3 Tag total size: %d\n", metadata.tagSize))

	return sb.String()
}

// GetInfo takes a reader, then returns metadata of the MP3, includes estimated duration
// If the data doesn't seem like an MP3, it returns an error
//
// totalSize is int64 to align with FileInfo.Size() and http.Response.ContentLength
func GetInfo(r io.Reader, totalSize int64) (*Metadata, error) {
	var metadata Metadata
	var err error

	skipReader := id3.NewSkipReader(r)
	metadata.tagSize, err = skipReader.ReadThrough()

	if err != nil {
		return &metadata, err
	}

	// Read MP3 frame header
	var headerBits uint32

	if err = binary.Read(r, binary.BigEndian, &headerBits); err != nil {
		return &metadata, err
	}

	if metadata.mp3Header, err = mp3header.Parse(headerBits); err != nil {
		return &metadata, err
	}

	metadata.calculateDuration(totalSize)

	return &metadata, nil
}
