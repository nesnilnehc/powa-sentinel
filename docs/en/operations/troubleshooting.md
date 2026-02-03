# Workflow Troubleshooting

Diagnose and fix CI/CD failures (CI on main/PR, Release on tag push).

**Audience:** Maintainers. For release details, see [Release Workflow](release-workflow.md). For deployment, see [Deployment](../guides/deployment.md).

## Quick Diagnosis

```
Workflow Failed?
├── Authentication Error → Permissions
├── Build Error → Build Issues
├── Registry Error → Registry Issues
├── Multi-arch Error → Platform Issues
└── Timeout Error → Performance Issues
```

## Authentication

### "failed to login to ghcr.io"

- Ensure **Settings → Actions → General** → "Read and write permissions"
- Ensure workflow has `packages: write`
- Forks cannot publish to main repo

### "Registry authentication failed after retry"

1. Set workflow permissions to read-write
2. Optional: add `GHCR_TOKEN` secret (PAT with `write:packages`)

## Build

### "failed to build docker image"

- Test locally: `docker build --platform linux/amd64 .`
- Check Dockerfile and `.dockerignore`
- Ensure `go.mod` and `go.sum` are committed

## Registry

### "failed to push to registry"

- Check [GitHub Status](https://status.github.com)
- Test auth: `echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin`

## Multi-arch

### "QEMU emulation setup failed"

- Retry (transient)
- Or temporarily use `platforms: linux/amd64` only

## Debug

- Add secrets: `ACTIONS_STEP_DEBUG=true`, `ACTIONS_RUNNER_DEBUG=true`
- Test locally: [act](https://github.com/nektos/act)
- Lint: `actionlint .github/workflows/*.yml`
