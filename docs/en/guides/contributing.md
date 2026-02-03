# Contributing

This guide covers **building, testing, and running** powa-sentinel locally. For deployment, see [Deployment](deployment.md).

## Build

### Makefile

```makefile
.PHONY: build clean test

BINARY_NAME=powa-sentinel

build:
	CGO_ENABLED=0 go build -o bin/$(BINARY_NAME) ./cmd/powa-sentinel

clean:
	rm -rf bin/

test:
	go test ./...
```

### Cross-compile (Linux)

```bash
GOOS=linux GOARCH=amd64 make build
```

## Test

### Unit tests

```bash
go test -v ./internal/...
```

### Integration tests

Requires a running PoWA repository database (PostgreSQL with PoWA and, for single-server, the monitored instance is the same). For example: testcontainers or docker-compose.

```bash
go test -tags=integration ./tests/...
```

## Run locally

```bash
# From source
go run ./cmd/powa-sentinel -config config/config.yaml.example

# Or built binary
./bin/powa-sentinel -config config/config.yaml.example
```

## Architecture

- [Architecture](../reference/architecture.md) — System design and project layout
- [PoWA Schema](../reference/powa-schema.md) — Data model and views used

## Release

See [Release Workflow](../operations/release-workflow.md) for how releases are produced.
