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

var errInvalidInput = fmt.Errorf("first argument must be a path or HTTP URL")

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

func processInput(location *url.URL) (*mp3len.Metadata, error) {
	var r io.ReadCloser
	var totalLength int64
	var err error

	if location.Scheme == "http" || location.Scheme == "https" {
		r, totalLength, err = openHTTP(location)
	} else if location.Scheme == "file" || location.Scheme == "" {
		r, totalLength, err = openFile(location)
	} else {
		return nil, errInvalidInput
	}

	if err != nil {
		return nil, err
	}

	defer r.Close()

	return mp3len.GetInfo(r, totalLength)
}

func main() {
	verbose := flag.Bool("verbose", false, "show verbose info such as id3 tags and mp3 format")

	flag.Parse()

	location, err := url.Parse(flag.Arg(0))

	if location == nil || location.Path == "" || err != nil {
		fmt.Fprintln(os.Stderr, errInvalidInput)
		os.Exit(1)
	}

	info, err := processInput(location)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(info.String(*verbose))
}
