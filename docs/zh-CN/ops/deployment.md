# 部署与运维指南

## 1. 构建策略

我们使用 `make` 来标准化构建命令。

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

### 1.2 二进制构建
对于 Linux 服务器（数据库环境常用）：
```bash
GOOS=linux GOARCH=amd64 make build
```

## 2. 测试策略

### 2.1 单元测试
运行标准 Go 测试以进行逻辑验证（规则引擎、配置加载）。
```bash
go test -v ./internal/...
```

### 2.2 集成测试
使用 `testcontainers-go` 或 `docker-compose` 启动临时的 PostgreSQL+PoWA 实例。
*   **目标**: 验证针对活动 `powa` 模式的 SQL 查询。
*   **命令**: `go test -tags=integration ./tests/...`

## 3. 分发

### 3.1 Docker 镜像
我们使用多阶段构建来保持镜像最小化（Distroless 或 Alpine）。

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

### 3.2 二进制文件
使用 **GoReleaser** 进行自动化 GitHub 发布（Linux/Mac/Windows 的 tar 包）。

## 4. 部署

### 4.1 独立部署 (Systemd)
适用于运行 PostgreSQL 的 VM/裸机。

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
适用于与 `powa-web` 一起作为 Sidecar 部署。

```yaml
version: '3'
services:
  powa-sentinel:
    image: powa-sentinel:latest
    volumes:
      - ./config.yaml:/config.yaml
    restart: unless-stopped
    depends_on:
      - postgres # PoWA 仓库数据库
```

### 4.3 Kubernetes
作为 Deployment 部署。建议：
*   **ConfigMap**: 挂载 `config.yaml`。
*   **资源**: 低占用（例如，100m CPU, 128Mi 内存）。
*   **安全**: 以只读文件系统、非 root 用户运行。
*   **可观测性**:
    *   `livenessProbe`: `httpGet` 路径 `/healthz` 端口 8080。
    *   `readinessProbe`: `httpGet` 路径 `/healthz` 端口 8080。
