# 部署指南

本文说明如何 **部署** powa-sentinel。构建与测试见 [贡献指南](contributing.md)。

## 分发

预构建镜像：[GitHub Container Registry](https://github.com/nesnilnehc/powa-sentinel/pkgs/container/powa-sentinel)

```
ghcr.io/nesnilnehc/powa-sentinel:latest
ghcr.io/nesnilnehc/powa-sentinel:v0.1.0
```

二进制压缩包：[GitHub Releases](https://github.com/nesnilnehc/powa-sentinel/releases)

## 独立部署 (Systemd)

适用于运行 PoWA 仓库数据库（或被监控 PostgreSQL）的 VM 或裸机：

`/etc/systemd/system/powa-sentinel.service`：

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

## Docker Compose

适用于与 powa-web 并行的 sidecar 部署：

```yaml
version: '3'
services:
  powa-sentinel:
    image: ghcr.io/nesnilnehc/powa-sentinel:latest
    volumes:
      - ./config.yaml:/config.yaml
    restart: unless-stopped
    depends_on:
      - postgres
```

## Kubernetes

以 Deployment 形式部署。镜像使用 `ghcr.io/nesnilnehc/powa-sentinel:latest` 或版本 tag。

- **ConfigMap**：挂载 `config.yaml`
- **资源**：低占用（如 100m CPU、128Mi 内存）
- **安全**：只读文件系统、非 root 用户
- **探针**：`livenessProbe` / `readinessProbe`：`httpGet` 路径 `/healthz`，端口 8080
