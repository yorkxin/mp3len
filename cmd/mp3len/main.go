package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/yorkxin/mp3len"
)

var errInvalidInput = fmt.Errorf("First argument must be a path or HTTP URL")

func openFile(location *url.URL) (io.ReadCloser, int64, error) {
	f, err := os.Open(location.Path)

	if err != nil {
		return nil, 0, err
	}

	stat, err := f.Stat()

	if err != nil {
		return nil, 0, err
	}

	length := stat.Size()

	return f, length, nil
}

func openHTTP(location *url.URL) (io.ReadCloser, int64, error) {
	req, err := http.NewRequest("GET", location.String(), nil)
	if err != nil {
		return nil, 0, err
	}

	// avoid reading the whole response from server
	// NOTE: this depends on server implementation of Range request.
	// TODO: find a better way to stream, rather than guessing length.
	req.Header.Set("Range", "bytes=0-65600")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, 0, err
	}

	length := resp.ContentLength

	if cr := resp.Header.Get("Content-Range"); cr != "" {
		// TODO: handle unit != "bytes"
		// TODO: handle unknown size
		splitted := strings.SplitN(cr, "/", 2)

		if len(splitted) == 2 {
			totalBytes, err := strconv.Atoi(splitted[1])
			if err == nil {
				length = int64(totalBytes)
			}
			if err != nil {
				// edge case: Content-Range is "*" or something unparsable
			}
		} else {
			// edge case: Content-Range is not in standard "unit start-end/total" foramt.
			// fallback to non-range request
		}
	}

	return resp.Body, length, nil
}

func main() {
	pause := flag.Bool("pause", false, "press any key to exit; useful for checking IO activity.")
	flag.Parse()

	input, err := url.Parse(flag.Arg(0))

	if input.Path == "" || err != nil {
		fmt.Fprintln(os.Stderr, errInvalidInput)
		os.Exit(1)
	}

	var r io.ReadCloser
	var totalLength int64

	if input.Scheme == "http" || input.Scheme == "https" {
		r, totalLength, err = openHTTP(input)
	} else if input.Scheme == "file" || input.Scheme == "" {
		r, totalLength, err = openFile(input)
	} else {
		fmt.Fprintln(os.Stderr, errInvalidInput)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer r.Close()

	duration, err := mp3len.EstimateDuration(r, totalLength)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", duration)

	if *pause == true {
		anyKeyReader := bufio.NewReader(os.Stdin)
		fmt.Printf("Press enter key to exit...")
		buf := make([]byte, 1)
		anyKeyReader.Read(buf)
	}
}
