VERSION ?= $(shell git describe --tags --always --dirty)
BRANCH  ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT  ?= $(shell git rev-parse --short HEAD)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')

BINARY_NAME := $(notdir $(shell go list -m))
LDFLAGS := -s -w \
           -X 'main.Version=$(VERSION)' \
           -X 'main.GitBranch=$(BRANCH)' \
           -X 'main.GitCommit=$(COMMIT)' \
           -X 'main.BuildTime=$(BUILD_TIME)'

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)

test:
	go test ./...
