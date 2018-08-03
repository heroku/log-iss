.PHONY = default build push update-deps

UBUNTU_VERSION := 18.04
GOLANG_VERSION := 1.10
VERSION=$(shell git log --pretty=format:'%h' -n 1)
IMAGE = heroku/log-iss
ECR_IMAGE =

default:
	# task def
	# --------
	# build: builds docker iamge
	# push:  builds and pushes docker image to ecr
	# test:  return 0

build: update-deps
	docker build -t $(IMAGE):$(VERSION) etc/docker
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

push: build
	bash etc/bin/ecr.sh push $(IMAGE) $(VERSION)

update-deps:
	docker pull golang:$(GOLANG_VERSION)
	docker pull ubuntu:$(UBUNTU_VERSION)

test:
	@true
