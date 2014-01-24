#!/usr/bin/env make -f

VERSION := $(shell git tag | tail -n 1 | cut -d v -f 2)

tempdir        := $(shell mktemp -d)
controldir     := $(tempdir)/DEBIAN
installpath    := $(tempdir)/usr/bin

define DEB_CONTROL
Package: log-iss
Version: $(VERSION)
Architecture: amd64
Maintainer: "Dan Peterson" <dan@heroku.com>
Section: heroku
Priority: optional
Description: Move logs from the Dyno to the Logplex.
endef
export DEB_CONTROL

deb: bin/log-iss
	mkdir -p -m 0755 $(controldir)
	echo "$$DEB_CONTROL" > $(controldir)/control
	mkdir -p $(installpath)
	install bin/log-iss $(installpath)/log-iss
	fakeroot dpkg-deb --build $(tempdir) .
	rm -rf $(tempdir)

bin/log-iss:
	git clone git://github.com/kr/heroku-buildpack-go.git .build
	.build/bin/compile . .build/cache/

clean:
	rm -rf bin
	rm -rf .build/
	rm -f log-iss*.deb

build: bin/log-iss
