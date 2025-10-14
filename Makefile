MODULE := github.com/cdinu/myuplink-naive-client
BIN := bin/nibe-fetch

.PHONY: all fmt lint test build clean

all: fmt lint test

fmt:
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -w ./ || gofmt -w ./cmd ./internal

lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --config=.golangci.yml

test:
	go test ./...

build:
	go build -o $(BIN) ./cmd/nibe-fetch

clean:
	rm -rf $(BIN) dist
