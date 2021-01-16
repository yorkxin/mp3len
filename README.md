# mp3len

Estimate MP3 duration by reading the first few KB's of the input.

## CLI Usage

```sh
$ go run cmd/mp3len/main.go <file|url>
```

For example:

```sh
$ go run cmd/mp3len/main.go ~/Downloads/file.mp3
49m17.122s
```

```sh
$ go run cmd/mp3len/main.go https://d1nz8yczgon8of.cloudfront.net/episodes/EP10.mp3
49m17.122s
```

## Why

The duration of MP3 file is required in some scenarios such as podcast RSS.
However that information is usually not available from ID3 tag. This process
is usually done by manual input, which is not good for automated process.

The article *[How to Get the Duration of a Remote MP3 File][1]* shows an idea
about how to *estimate* MP3 duration by reading through the MP3 frame headers.
In the best case we only need to read the first few KBs (under 100). Note that
it assumes that the bit rate does not change for the whole file.

## TODO

* More Unit Tests
* JSON Output
* More efficient read-skip: all the way to the MP3 frame sync bits, ignoring
  all ID3 tags.
* S3 Support
* Lambda Example
* Distribute this module
* Avoid reading too much data from HTTP (currently it reads a few MBs)

[1]:https://www.factorialcomplexity.com/blog/how-to-get-a-duration-of-a-remote-mp3-file

## License

MIT. See [LICENSE][./LICENSE]
