GO_LINKER_SYMBOL := "main.version"
GO_LINKER_VALUE := $(shell git describe --tags --always | sed s/^v//)

all: test

test:
	go test -v ./cmd/...

bench:
	go test -v -bench=. ./cmd/...

install:
	go install -a -ldflags "-X ${GO_LINKER_SYMBOL}=${GO_LINKER_VALUE}" ./cmd/...

build:
	go build -a -ldflags "-X ${GO_LINKER_SYMBOL}=${GO_LINKER_VALUE}" ./cmd/...