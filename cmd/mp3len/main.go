package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

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

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.ContentLength, nil
}

func processInput(location *url.URL) (time.Duration, error) {
	var r io.ReadCloser
	var totalLength int64
	var err error

	if location.Scheme == "http" || location.Scheme == "https" {
		r, totalLength, err = openHTTP(location)
	} else if location.Scheme == "file" || location.Scheme == "" {
		r, totalLength, err = openFile(location)
	} else {
		return 0, errInvalidInput
	}

	if err != nil {
		return 0, err
	}

	defer r.Close()

	return mp3len.EstimateDuration(r, totalLength)
}

func main() {
	pause := flag.Bool("pause", false, "press any key to exit; useful for checking IO activity.")
	flag.Parse()

	location, err := url.Parse(flag.Arg(0))

	if location.Path == "" || err != nil {
		fmt.Fprintln(os.Stderr, errInvalidInput)
		os.Exit(1)
	}

	duration, err := processInput(location)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(duration.String())

	if *pause == true {
		anyKeyReader := bufio.NewReader(os.Stdin)
		fmt.Printf("Press enter key to exit...")
		buf := make([]byte, 1)
		anyKeyReader.Read(buf)
	}
}
