BINARY = bin/mp3len

.PHONY: all lint build test clean

all: lint test build

test:
	go test ./...

build:
	go build -o ${BINARY} cmd/mp3len/main.go

clean:
	if [ -f "${BINARY}" ]; then rm "${BINARY}"; fi

lint:
	golangci-lint run
