.PHONY: build clean test lint docker run help

# Binary name
BINARY_NAME=powa-sentinel

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Default target
all: build

## help: Show this help message
help:
	@echo "powa-sentinel build targets:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary for current platform
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/powa-sentinel

## build-linux: Build for Linux (amd64)
build-linux: build-linux-amd64

## build-linux-amd64: Build for Linux (amd64)
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/powa-sentinel

## build-linux-arm64: Build for Linux (arm64)
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/powa-sentinel

## build-darwin: Build for macOS (universal/both)
build-darwin: build-darwin-amd64 build-darwin-arm64

## build-darwin-amd64: Build for macOS (intel)
build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/powa-sentinel

## build-darwin-arm64: Build for macOS (apple silicon)
build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/powa-sentinel

## build-windows: Build for Windows (amd64)
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/powa-sentinel

## build-all: Build for all supported platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows

## clean: Remove build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

## test: Run all tests
test:
	go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## mod: Tidy go modules
mod:
	go mod tidy
	go mod verify

## docker: Build Docker image (local architecture)
docker:
	docker buildx build --load -t powa-sentinel:$(VERSION) -t powa-sentinel:latest .

## docker-push: Build and push multi-arch Docker image (requires REGISTRY env var)
docker-push:
	@if [ -z "$(REGISTRY)" ]; then echo "REGISTRY not set"; exit 1; fi
	docker buildx build --platform linux/amd64,linux/arm64 --push \
		-t $(REGISTRY)/powa-sentinel:$(VERSION) \
		-t $(REGISTRY)/powa-sentinel:latest .

## run: Run locally with example config
run: build
	./bin/$(BINARY_NAME) -config config/config.yaml.example

## run-once: Run analysis once and exit
run-once: build
	./bin/$(BINARY_NAME) -config config/config.yaml.example -once

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

## release: Create release tag (usage: make release RELEASE_VERSION=v0.2.0)
release:
	@if [ -z "$(RELEASE_VERSION)" ]; then echo "Usage: make release RELEASE_VERSION=v0.2.0"; exit 1; fi
	@echo "Creating tag $(RELEASE_VERSION)..."
	git tag -a $(RELEASE_VERSION) -m "Release $(RELEASE_VERSION)"
	@echo ""
	@echo "Tag created. Next steps:"
	@echo "  1. Push tag:    git push origin $(RELEASE_VERSION)"
	@echo "  2. (Optional) Create GitHub Release and/or run: goreleaser release --clean"
	@echo "  See docs/en/operations/release-workflow.md (en) or docs/zh-CN/operations/release-workflow.md (zh-CN) for details."
