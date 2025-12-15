# Deployment & Operations Guide

## 1. Build Strategy

We use `make` to standardize build commands.

### 1.1 Makefile
```makefile
.PHONY: build clean test docker

BINARY_NAME=powa-sentinel

build:
	CGO_ENABLED=0 go build -o bin/$(BINARY_NAME) ./cmd/powa-sentinel

clean:
	rm -rf bin/

test:
	go test ./...
```

### 1.2 Binary Build
For Linux servers (common for DB environments):
```bash
GOOS=linux GOARCH=amd64 make build
```

## 2. Testing Strategy

### 2.1 Unit Tests
Run standard Go tests for logic verification (Rule Engine, Config loading).
```bash
go test -v ./internal/...
```

### 2.2 Integration Tests
Use `testcontainers-go` or `docker-compose` to spin up a ephemeral PostgreSQL+PoWA instance.
*   **Goal**: Verify SQL queries against active `powa` schema.
*   **Command**: `go test -tags=integration ./tests/...`

## 3. Distribution

### 3.1 Docker Images
We use a multi-stage build to keep the image minimal (Distroless or Alpine).

**Dockerfile**:
```dockerfile
# Stage 1: Build
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o powa-sentinel ./cmd/powa-sentinel

# Stage 2: Runtime
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/powa-sentinel /
COPY config/config.yaml.example /config.yaml
USER nonroot:nonroot
ENTRYPOINT ["/powa-sentinel"]
```

### 3.2 Binaries
Use **GoReleaser** for automated GitHub Releases (tarballs for Linux/Mac/Windows).

## 4. Deployment

### 4.1 Standalone (Systemd)
Suitable for VM/Bare-metal where PostgreSQL runs.

`/etc/systemd/system/powa-sentinel.service`:
```ini
[Unit]
Description=PoWA Sentinel Push Service
After=network.target postgresql.service

[Service]
Type=simple
User=postgres
ExecStart=/usr/local/bin/powa-sentinel -config /etc/powa-sentinel/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### 4.2 Docker Compose
Suitable for sidecar deployment alongside `powa-web`.

```yaml
version: '3'
services:
  powa-sentinel:
    image: powa-sentinel:latest
    volumes:
      - ./config.yaml:/config.yaml
    restart: unless-stopped
    depends_on:
      - postgres # The PoWA repository database
```

### 4.3 Kubernetes
Deploy as a Deployment. Recommendations:
*   **ConfigMap**: Mount `config.yaml`.
*   **Resources**: Low footprint (e.g., 100m CPU, 128Mi Memory).
*   **Security**: Run as ReadOnly filesystem, non-root user.
*   **Observability**:
    *   `livenessProbe`: `httpGet` path `/healthz` port 8080.
    *   `readinessProbe`: `httpGet` path `/healthz` port 8080.

