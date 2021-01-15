package mp3len

import (
	"fmt"
	"io"
	"time"

	"github.com/yorkxin/mp3len/id3"
	"github.com/yorkxin/mp3len/mpeg"
)

// EstimateDuration takes a reader, then returns duration of the MP3 in seconds.
// If the data doesn't seem like an MP3, it returns an error
func EstimateDuration(r io.Reader, totalSize int64) (duration time.Duration, err error) {
	// Algorithm from https://www.factorialcomplexity.com/blog/how-to-get-a-duration-of-a-remote-mp3-file
	id3Size, _, err := id3.ReadFrames(r)

	if err != nil {
		return 0, err
	}

	// read MP3 frame header; first 11 bytes should be all 1
	mp3HeaderBytes := make([]byte, 4)
	_, err = r.Read(mp3HeaderBytes)

	if err != nil {
		return 0, fmt.Errorf("unable to read after id3tags")
	}

	mp3Header, _ := mpeg.ParseMP3Header(mp3HeaderBytes)

	duration = time.Duration((totalSize - int64(id3Size)) / (int64(mp3Header.BitRate) / 8) * 1000000)

	if mp3Header.ChannelMode == mpeg.ChannelModeMono {
		duration *= 2
	}

	return
}
