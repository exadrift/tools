NAME := $(shell basename $${PWD})
GO_VERSION := 1.26
LINTER_VERSION := 2.12.2
VERSION ?= local
GO_IMAGE := golang:$(GO_VERSION)-alpine
MOUNT ?=
DOCKER_CMD := docker run --network host --rm -e CGO_ENABLED=0 -e HOME=$$HOME -v $$HOME:$$HOME -v $(shell pwd):/build -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock $(MOUNT) -w /build
GO_RUN_USER := $(DOCKER_CMD) -u $(shell id -u):$(shell id -g) $(GO_IMAGE) go
GO_RUN_ROOT := $(DOCKER_CMD) $(GO_IMAGE) go
GO_FILES := $(shell find . -type f -path **/*.go -not -path "./vendor/*")
IMAGE_NAME := exadrift/$(NAME)
IMAGE_TAG := $(IMAGE_NAME):$(VERSION)

.PHONY: lint-check
lint-check:
	docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v$(LINTER_VERSION) golangci-lint run

.PHONY: test
test:
	# tests are run as root for docker socket access within the container
	$(GO_RUN_ROOT) test -cover -p 1 --timeout 10m ./...