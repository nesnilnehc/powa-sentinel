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
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/powa-sentinel

## build-darwin: Build for macOS (arm64)
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/powa-sentinel

## build-all: Build for all platforms
build-all: build-linux build-darwin

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
