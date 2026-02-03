# 发布流程

统一发布流程：构建、发布 Docker 镜像到 GHCR、创建二进制包与 GitHub Release。

**面向**：维护者。部署见 [部署指南](../guides/deployment.md)，Workflow 故障见 [故障排查](troubleshooting.md)。

## 概述

推送版本 tag（`v*`）时，单一 workflow（`.github/workflows/release.yml`）运行，通过 GoReleaser 完成：

1. 多平台二进制构建（linux/darwin/windows，amd64/arm64）
2. 多架构 Docker 镜像构建并发布到 GHCR
3. 创建 GitHub Release（含二进制压缩包与校验和）

## 触发条件

| 事件 | 动作 |
|------|------|
| 推送 tag `v*` | 完整发布：GHCR 镜像 + GitHub Release + 二进制 |

## 镜像仓库

- **Registry**：`ghcr.io`
- **仓库**：`nesnilnehc/powa-sentinel`
- **镜像路径**：`ghcr.io/nesnilnehc/powa-sentinel`

## 镜像标签

对于 tag `v1.0.0`：

```
ghcr.io/nesnilnehc/powa-sentinel:v1.0.0
ghcr.io/nesnilnehc/powa-sentinel:latest
```

## 步骤

1. Checkout（完整历史）
2. 配置 Go 与模块缓存
3. 配置 QEMU 与 Docker Buildx 以支持多架构
4. 登录 GHCR（`GHCR_TOKEN` 或 `GITHUB_TOKEN`）
5. 执行 GoReleaser：`goreleaser release --clean`

## 配置文件

- **Workflow**：`.github/workflows/release.yml`
- **GoReleaser**：`.goreleaser.yaml`
- **Dockerfile**：`Dockerfile`

## CI 与 CD

- **CI**（`ci.yml`）：`main` 的 `push`、`pull_request`。构建、测试、govulncheck、Trivy。**不**发布。
- **CD**（`release.yml`）：仅 `v*` tag 的 `push`。发布镜像与 Release。

## 验证发布镜像

```bash
docker pull ghcr.io/nesnilnehc/powa-sentinel:latest
docker pull ghcr.io/nesnilnehc/powa-sentinel:v0.1.0
docker inspect ghcr.io/nesnilnehc/powa-sentinel:latest | jq '.[0].Config.Labels'
```
