# Release Workflow

Unified release workflow: build, publish Docker images to GHCR, create binary packages and GitHub Releases.

**Audience:** Maintainers. For deployment, see [Deployment](../guides/deployment.md). For workflow failures, see [Troubleshooting](troubleshooting.md).

## Overview

A single workflow (`.github/workflows/release.yml`) runs when a version tag (`v*`) is pushed. It uses GoReleaser to:

1. Build binaries for multiple platforms (linux/darwin/windows, amd64/arm64)
2. Build and publish multi-architecture Docker images to GHCR
3. Create GitHub Release with binary archives and checksums

## Triggers

| Event | Action |
|-------|--------|
| `push` tag `v*` | Full release: GHCR images + GitHub Release + binaries |

## Registry

- **Registry:** `ghcr.io`
- **Repository:** `nesnilnehc/powa-sentinel`
- **Image path:** `ghcr.io/nesnilnehc/powa-sentinel`

## Image Tags

For tag `v1.0.0`:

```
ghcr.io/nesnilnehc/powa-sentinel:v1.0.0
ghcr.io/nesnilnehc/powa-sentinel:latest
```

## Steps

1. Checkout (full history)
2. Set up Go with module cache
3. Set up QEMU and Docker Buildx for multi-arch
4. Log in to GHCR (`GHCR_TOKEN` or `GITHUB_TOKEN`)
5. Run GoReleaser: `goreleaser release --clean`

## Configuration

- **Workflow:** `.github/workflows/release.yml`
- **GoReleaser:** `.goreleaser.yaml`
- **Dockerfile:** `Dockerfile`

## CI vs CD

- **CI** (`ci.yml`): `push` to `main`, `pull_request`. Build, test, govulncheck, Trivy. Does **not** publish.
- **CD** (`release.yml`): `push` tag `v*` only. Publishes images and releases.

## Verify Published Images

```bash
docker pull ghcr.io/nesnilnehc/powa-sentinel:latest
docker pull ghcr.io/nesnilnehc/powa-sentinel:v0.1.0
docker inspect ghcr.io/nesnilnehc/powa-sentinel:latest | jq '.[0].Config.Labels'
```
