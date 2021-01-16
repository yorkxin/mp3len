package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

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

func main() {
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
}
