SHELL := /bin/bash

IMAGE_NAME := daviuefs/test-client

.PHONY: build pull run

build:
	docker build -t $(IMAGE_NAME) -f dockerfile ..

pull:
	docker pull $(IMAGE_NAME)

run:
	docker run -it --rm --name test-client $(IMAGE_NAME)
