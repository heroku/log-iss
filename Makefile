UBUNTU_VERSION := 18.04
GOLANG_VERSION := 1.10
VERSION=$(shell git log --pretty=format:'%h' -n 1)
IMAGE = heroku/log-iss
CURRENT_BRANCH := $(shell git branch | grep "^*" | awk '{ print $$NF }')

bin/forwarder:
	go build -o bin/forwarder ./...

clean:
	rm -f bin/forwarder

docker/build: update-deps
	docker build -t $(IMAGE):$(VERSION) .
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

push: docker/build
	bash bin/ecr.sh push $(IMAGE) $(VERSION)

update-deps:
	docker pull golang:$(GOLANG_VERSION)
	docker pull ubuntu:$(UBUNTU_VERSION)

test:
	true

build:
	# Build master branch
	aws codebuild start-build --project-name heroku-log-iss

build/branch:
	# Build $(CURRENT_BRANCH) branch
	aws codebuild start-build --project-name heroku-log-iss --source-version $(CURRENT_BRANCH)

.PHONY: clean build push update-deps test docker/build
