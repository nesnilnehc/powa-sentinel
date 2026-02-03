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

## PoWA repository log warnings

If you see warnings like:

- **"pg_stat_kcache extension present but no history table found in PoWA 4"**  
  The kcache history table was not found. Ensure you ran `SELECT powa_kcache_register();` after installing `pg_stat_kcache`, and that powa-archivist has created the table (usually in the `powa` schema). Sentinel looks for tables named like `powa_%kcache%history` in the `public` and `powa` schemas.

- **"powa_qualstats_indexes view does not exist, skipping index suggestions"**  
  The qualstats integration is not fully set up. Run `SELECT powa_qualstats_register();` on the PoWA database (as superuser) so that PoWA creates the required views. If you use a custom PoWA setup, ensure the view exists and the read-only user has `SELECT` on it.

These warnings do not stop analysis: slow-query and regression checks still run; only kcache-based enrichment and index suggestions are skipped.

### "undefined_table" or "undefined_column" errors

If the PoWA repository database (the database powa-sentinel connects to) returns errors about an undefined table or column (e.g. when querying `powa_statements_history` or other PoWA objects), check that your **PoWA (powa-archivist) version** is in the [supported compatibility matrix](../reference/compatibility.md). Unsupported or mismatched versions use different schemas and will cause these errors. Ensure your connection `search_path` includes the schema where PoWA created its views (often `powa` or `public`).
