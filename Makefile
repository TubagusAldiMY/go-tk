BINARY_NAME=go-tk
MODULE=github.com/TubagusAldiMY/go-tk
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X $(MODULE)/cmd/go-tk/build.Version=$(VERSION) -X $(MODULE)/cmd/go-tk/build.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) -X $(MODULE)/cmd/go-tk/build.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: all build test lint clean install fmt vet check coverage snapshot release tidy

all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/go-tk/

install:
	go install $(LDFLAGS) ./cmd/go-tk/

test:
	go test -v -race -coverprofile=coverage.out ./...

test-short:
	go test -short ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

snapshot:
	goreleaser build --snapshot --clean

release:
	goreleaser release --clean

tidy:
	go mod tidy

# Pre-commit check (AGENTS.md Section 8.4 & 14.9)
# Run this before every commit — fail fast if any check fails
check: fmt vet test
	@echo ""
	@echo "✅ All pre-commit checks passed!"
	@echo "   - Code formatting: OK"
	@echo "   - Go vet: OK"  
	@echo "   - Tests (with race detector): OK"
	@echo ""
	@echo "Ready to commit. Use: git add -A && git commit"

.DEFAULT_GOAL := build
