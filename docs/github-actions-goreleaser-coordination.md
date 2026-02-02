# GitHub Actions and GoReleaser Coordination

This document describes how the GitHub Actions Docker publishing workflow coordinates with GoReleaser to prevent conflicts and ensure consistent Docker image publishing.

## Overview

Both GitHub Actions and GoReleaser are configured to publish Docker images to GitHub Container Registry (ghcr.io). To prevent conflicts and ensure consistent behavior, they coordinate through the following mechanisms:

## Coordination Strategy

### 1. Event-Based Coordination

**GitHub Actions Workflow Triggers:**
- `push` to `main` branch → Publishes `main` and `main-<sha>` tags
- `pull_request` → Builds but does NOT publish (validation only)
- `push` of Git tags (`v*`) → Publishes version tags and `latest`
- `release` published → Publishes version tags and `latest`

**GoReleaser Triggers:**
- Manual execution on tagged releases → Publishes `{{ .Tag }}` and `latest` tags
- Typically used for official releases with additional artifacts (binaries, archives)

### 2. Registry and Repository Consistency

Both tools use identical configuration:
- **Registry:** `ghcr.io`
- **Repository:** `powa-team/powa-sentinel`
- **Dockerfile:** `Dockerfile` (same file)
- **Build Arguments:** `VERSION`, `COMMIT`, `BUILD_DATE`

### 3. Tagging Strategy Coordination

| Scenario | GitHub Actions Tags | GoReleaser Tags | Conflict Resolution |
|----------|-------------------|-----------------|-------------------|
| Push to main | `main`, `main-<sha>` | None | No conflict |
| Pull request | None (build only) | None | No conflict |
| Git tag push | `v1.0.0`, `latest` | None (unless manually triggered) | Registry handles duplicates |
| Release published | `v1.0.0`, `latest` | `v1.0.0`, `latest` | Registry handles duplicates |

### 4. Build Argument Consistency

Both tools use the same build arguments with equivalent values:

**GitHub Actions:**
```yaml
build-args: |
  VERSION=${{ steps.build-meta.outputs.version }}
  COMMIT=${{ steps.build-meta.outputs.commit }}
  BUILD_DATE=${{ steps.build-meta.outputs.build_date }}
```

**GoReleaser:**
```yaml
build_flag_templates:
  - "--build-arg=VERSION={{.Version}}"
  - "--build-arg=COMMIT={{.ShortCommit}}"
  - "--build-arg=BUILD_DATE={{.Date}}"
```

## Conflict Prevention

### 1. Registry-Level Deduplication

GitHub Container Registry handles duplicate tags gracefully:
- Publishing the same tag multiple times overwrites the previous image
- No errors occur when both tools publish identical tags
- Latest successful publish wins

### 2. Conditional Publishing Logic

GitHub Actions workflow includes conditional publishing:
```yaml
# Only publish for main branch and tags, not for pull requests
push: ${{ steps.build-meta.outputs.should_publish == 'true' }}
```

### 3. Repository Validation

Both tools validate repository ownership:
- GitHub Actions: `github.repository == 'powa-team/powa-sentinel'`
- GoReleaser: `release.github.owner: powa-team` and `release.github.name: powa-sentinel`

## Recommended Usage Patterns

### For Development Workflow

1. **Pull Requests:** GitHub Actions builds and validates (no publishing)
2. **Main Branch:** GitHub Actions publishes `main` and `main-<sha>` tags
3. **Manual Testing:** Use `main` tag for latest development version

### For Release Workflow

**Option A: GitHub Actions Only**
1. Create Git tag: `git tag v1.0.0 && git push origin v1.0.0`
2. GitHub Actions automatically publishes `v1.0.0` and `latest` tags
3. Create GitHub release from the tag (optional)

**Option B: GoReleaser Only**
1. Create Git tag: `git tag v1.0.0 && git push origin v1.0.0`
2. Run GoReleaser: `goreleaser release --clean`
3. GoReleaser creates GitHub release and publishes Docker images

**Option C: Coordinated (Recommended)**
1. Create Git tag: `git tag v1.0.0 && git push origin v1.0.0`
2. GitHub Actions publishes Docker images immediately
3. Run GoReleaser for additional artifacts: `goreleaser release --clean`
4. Both tools publish to same tags (registry handles duplicates)

## Monitoring and Troubleshooting

### Checking Published Images

```bash
# List all tags
docker pull ghcr.io/powa-team/powa-sentinel --all-tags

# Check specific tag
docker pull ghcr.io/powa-team/powa-sentinel:latest
docker inspect ghcr.io/powa-team/powa-sentinel:latest
```

### Verifying Build Metadata

```bash
# Check build arguments in image labels
docker inspect ghcr.io/powa-team/powa-sentinel:latest | jq '.[0].Config.Labels'

# Run container to check version
docker run --rm ghcr.io/powa-team/powa-sentinel:latest --version
```

### Common Issues

1. **Duplicate Publishing:** Not an error - registry handles gracefully
2. **Tag Conflicts:** Latest successful publish wins
3. **Build Argument Mismatch:** Tests validate consistency
4. **Repository Access:** Both tools validate repository ownership

## Testing Coordination

The coordination is validated through automated tests:

- `TestGoReleaserCoordination`: Validates configuration consistency
- `TestGoReleaserConflictPrevention`: Ensures proper conflict handling
- `TestDockerfileIntegration`: Verifies both tools use same Dockerfile

Run tests:
```bash
cd test
go test -v ./github-actions -run TestGoReleaser
```

## Configuration Files

- **GitHub Actions:** `.github/workflows/docker-publish.yml`
- **GoReleaser:** `.goreleaser.yaml`
- **Dockerfile:** `Dockerfile`
- **Coordination Tests:** `test/github-actions/goreleaser_coordination_test.go`