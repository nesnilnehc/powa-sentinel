# 贡献指南

本文说明如何 **本地构建、测试与运行** powa-sentinel。部署见 [部署指南](deployment.md)。

## 构建

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

### 交叉编译 (Linux)

```bash
GOOS=linux GOARCH=amd64 make build
```

## 测试

### 单元测试

```bash
go test -v ./internal/...
```

### 集成测试

需运行中的 PostgreSQL+PoWA 实例（如 testcontainers 或 docker-compose）：

```bash
go test -tags=integration ./tests/...
```

## 本地运行

```bash
# 从源码
go run ./cmd/powa-sentinel -config config/config.yaml.example

# 或已构建二进制
./bin/powa-sentinel -config config/config.yaml.example
```

## 架构

- [架构](../reference/architecture.md) — 系统设计与项目布局
- [PoWA Schema](../reference/powa-schema.md) — 使用的数据模型与视图

## 发布

发布流程见 [发布流程](../operations/release-workflow.md)。
