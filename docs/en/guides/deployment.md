# Deployment Guide

This guide covers **deploying** powa-sentinel. For build and test (contributors), see [Contributing](contributing.md).

## Distribution

Pre-built images: [GitHub Container Registry](https://github.com/nesnilnehc/powa-sentinel/pkgs/container/powa-sentinel)

```
ghcr.io/nesnilnehc/powa-sentinel:latest
ghcr.io/nesnilnehc/powa-sentinel:v0.1.0
```

Binary archives: [GitHub Releases](https://github.com/nesnilnehc/powa-sentinel/releases)

## Standalone (Systemd)

For VM or bare-metal where the PoWA repository database (or monitored PostgreSQL) runs:

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

## Docker Compose

For sidecar deployment alongside powa-web:

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

Deploy as a Deployment. Use image `ghcr.io/nesnilnehc/powa-sentinel:latest` or a version tag.

- **ConfigMap**: Mount `config.yaml`
- **Resources**: Low footprint (e.g. 100m CPU, 128Mi memory)
- **Security**: ReadOnly filesystem, non-root user
- **Probes**:
  - `livenessProbe` / `readinessProbe`: `httpGet` path `/healthz`, port 8080
