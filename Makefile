BIN_DIR         ?= $(shell go env GOPATH)/bin

default: test

deps: ## download go modules
	go mod download

fmt: # ensure consistent code style
	go run oss.indeed.com/go/go-groups -w .
	gofmt -s -w .

lint/install:
	@if ! golangci-lint --version > /dev/null 2>&1; then \
	  echo "Installing golangci-lint"; \
	  curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(BIN_DIR) v1.25.1; \
	fi

lint: lint/install ## run golangci-lint
	golangci-lint run

test: lint ## run go tests
	go vet ./...
	go test -race ./...

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
