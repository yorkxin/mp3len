package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yorkxin/mp3len"
)

func main() {
	flag.Parse()

	input := flag.Arg(0)

	// TODO: support http[s]:// or s3:// using interfaces
	if input == "" {
		fmt.Fprintln(os.Stderr, "first argument must be a filename")
		os.Exit(1)
	}

	f, err := os.Open(input)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open file in read mode")
		os.Exit(1)
	}

	// TODO: handle error
	stat, _ := f.Stat()
	duration, err := mp3len.EstimateDuration(f, stat.Size())

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", duration)
}
