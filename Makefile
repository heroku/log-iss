UBUNTU_VERSION := 18.04
GOLANG_VERSION := 1.10
VERSION=$(shell git log --pretty=format:'%h' -n 1)
IMAGE = heroku/log-iss

bin/forwarder:
	go build -o bin/forwarder ./...

clean:
	rm -f bin/forwarder

build: update-deps
	docker build -t $(IMAGE):$(VERSION) .
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

push: build
	bash bin/ecr.sh push $(IMAGE) $(VERSION)

update-deps:
	docker pull golang:$(GOLANG_VERSION)
	docker pull ubuntu:$(UBUNTU_VERSION)

test:
	true

.PHONY: clean build push update-deps test
