MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := all

BIN_DIR ?= $(shell go env GOPATH)/bin
export PATH := $(PATH):$(BIN_DIR)

.PHONY: deps
deps: ## download go modules
	go mod download

.PHONY: fmt
fmt: ## ensure consistent code style
	go run oss.indeed.com/go/go-groups@v1.1.3 -w .
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run --fix > /dev/null 2>&1 || true
	go mod tidy

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run
	@if [ -n "$$(go run oss.indeed.com/go/go-groups@v1.1.3 -l .)" ]; then \
		echo -e "\033[0;33mdetected fmt problems: run \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m"; \
		exit 1; \
	fi

.PHONY: test
test: lint ## run go tests
	go test ./... -race

.PHONY: build
build: ## compile and build artifact
	go build .

.PHONY: all
all: test build

.PHONY: help
help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
