GO_LINKER_SYMBOL := "main.version"

all: test

test:
	govendor test -v +local

bench:
	govendor test -v -bench=. +local

install:
	$(eval GO_LINKER_VALUE := $(shell git describe --tags --always | sed s/^v//))
	go install -a -ldflags "-X ${GO_LINKER_SYMBOL}=${GO_LINKER_VALUE}" ./...

update-deps: govendor
	govendor fetch +out

govendor:
	go get -u github.com/kardianos/govendor
