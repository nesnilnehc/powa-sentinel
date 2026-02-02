# GitHub Actions Docker Publishing Workflow

## Overview

This document provides comprehensive documentation for the GitHub Actions Docker publishing workflow that automatically builds and publishes multi-architecture Docker images to GitHub Container Registry (ghcr.io).

## Workflow Features

- **Multi-architecture builds**: Supports linux/amd64 and linux/arm64 platforms
- **Conditional publishing**: Publishes only for main branch and tags, validates PRs
- **Advanced caching**: Docker layer and Go module caching for optimal performance
- **Security-first**: Secure authentication with minimal permissions
- **Error handling**: Comprehensive error handling with retry logic
- **Registry validation**: Post-publish verification of image accessibility
- **Metadata compliance**: OCI-compliant image labels and metadata

## Repository Setup Requirements

### Required Repository Settings

1. **Repository Permissions**
   - Enable "Actions" in repository settings
   - Allow "Read and write permissions" for GITHUB_TOKEN
   - Enable "Allow GitHub Actions to create and approve pull requests" (optional)

2. **Package Settings**
   - Enable "Packages" in repository settings
   - Set package visibility to "Public" or "Private" as needed
   - Configure package permissions for the repository

### Required Workflow Permissions

The workflow requires these permissions in the workflow file:

```yaml
permissions:
  contents: read      # Required to read repository contents
  packages: write     # Required to publish to GitHub Container Registry
  id-token: write     # Required for OIDC token generation (future security enhancement)
```

### Repository Secrets

No additional secrets are required. The workflow uses the built-in `GITHUB_TOKEN` which is automatically provided by GitHub Actions.

## Workflow Triggers

### Automatic Triggers

| Event | Trigger | Action | Publishing |
|-------|---------|--------|------------|
| Push to main | `push: branches: [main]` | Build multi-arch images | ✅ Publish with `main` and `main-<sha>` tags |
| Pull Request | `pull_request: branches: [main]` | Build multi-arch images | ❌ Build only (validation) |
| Tag Push | `push: tags: ['v*']` | Build multi-arch images | ✅ Publish with version and `latest` tags |
| Release Published | `release: types: [published]` | Build multi-arch images | ✅ Publish with version and `latest` tags |

### Manual Triggers

The workflow can be manually triggered from the GitHub Actions tab:
1. Go to "Actions" tab in your repository
2. Select "Docker Build and Publish" workflow
3. Click "Run workflow"
4. Choose the branch and click "Run workflow"

## Image Tagging Strategy

### Tag Generation Logic

The workflow generates different tags based on the trigger event:

**For Git Tags (v1.0.0):**
```
ghcr.io/powa-team/powa-sentinel:v1.0.0
ghcr.io/powa-team/powa-sentinel:latest
```

**For Main Branch:**
```
ghcr.io/powa-team/powa-sentinel:main
ghcr.io/powa-team/powa-sentinel:main-<commit-sha>
```

**For Pull Requests (build only, no publishing):**
```
ghcr.io/powa-team/powa-sentinel:pr-<number>
```

### Image Metadata

All published images include OCI-compliant metadata labels:

```yaml
org.opencontainers.image.title: powa-sentinel
org.opencontainers.image.description: PostgreSQL Workload Analyzer Sentinel
org.opencontainers.image.vendor: powa-team
org.opencontainers.image.source: https://github.com/powa-team/powa-sentinel
org.opencontainers.image.version: <version>
org.opencontainers.image.revision: <commit-sha>
org.opencontainers.image.created: <build-timestamp>
```

## Build Process

### Multi-Architecture Support

The workflow builds images for multiple architectures using Docker Buildx:

1. **QEMU Setup**: Configures emulation for cross-platform builds
2. **Buildx Configuration**: Sets up Docker Buildx with container driver
3. **Platform Targets**: Builds for `linux/amd64` and `linux/arm64`
4. **Manifest Creation**: Creates multi-architecture manifest

### Build Arguments

The following build arguments are passed to the Dockerfile:

| Argument | Source | Example |
|----------|--------|---------|
| `VERSION` | Git tag or commit SHA | `v1.0.0` or `abc1234` |
| `COMMIT` | Git commit SHA | `abc1234567890abcdef` |
| `BUILD_DATE` | Build timestamp | `2024-01-15T10:30:00Z` |

### Caching Strategy

The workflow implements a multi-tier caching strategy:

**Docker Layer Caching:**
- Uses GitHub Actions cache for build layers
- Registry-based cache for cross-runner sharing
- Branch-specific cache scopes with fallbacks

**Go Module Caching:**
- Caches Go modules based on `go.sum` hash
- Shared across workflow runs for faster builds
- Automatic cache restoration and saving

## Security Features

### Authentication

- Uses GitHub's built-in `GITHUB_TOKEN` for registry authentication
- No manual credential configuration required
- Automatic token rotation and expiration handling

### Security Validations

1. **Repository Ownership**: Validates repository belongs to `powa-team`
2. **Fork Protection**: Prevents workflow execution on unauthorized forks
3. **Credential Security**: Ensures no credential exposure in logs
4. **Permission Validation**: Verifies minimum required permissions

### Security Best Practices

- Workflow runs only on main repository, not forks
- Uses minimal required permissions
- No secrets exposed in logs or outputs
- Automatic logout after registry operations

## Error Handling and Recovery

### Retry Logic

The workflow includes automatic retry for transient failures:

- **Checkout failures**: Automatic retry with extended timeout
- **QEMU setup failures**: Retry with different configuration
- **Buildx setup failures**: Retry with simplified settings
- **Authentication failures**: Retry with fresh token
- **Build failures**: Retry with simplified caching

### Error Analysis

When failures occur, the workflow provides detailed analysis:

- **Failure categorization**: Identifies the type of failure
- **Diagnostic information**: Provides relevant context and logs
- **Troubleshooting guidance**: Suggests resolution steps
- **Common issues**: Lists known problems and solutions

### Failure Scenarios

| Failure Type | Retry Behavior | Resolution |
|--------------|----------------|------------|
| Network timeout | Automatic retry | Extended timeout |
| Authentication failure | Retry with fresh token | Check permissions |
| Build failure | Retry with minimal cache | Check Dockerfile |
| Registry unavailable | Retry with backoff | Check registry status |

## Performance Optimization

### Build Time Optimization

- **Layer caching**: Reuses Docker layers across builds
- **Dependency caching**: Caches Go modules and build dependencies
- **Parallel builds**: Builds multiple architectures in parallel
- **Optimized context**: Uses `.dockerignore` to minimize build context

### Cache Effectiveness

The workflow monitors and reports cache effectiveness:

- **Cache hit rates**: Reports Go module and Docker layer cache hits
- **Build duration**: Tracks build times for performance monitoring
- **Cache size**: Monitors cache storage usage

### Performance Metrics

Typical build times (with warm cache):
- **Pull Request validation**: 3-5 minutes
- **Main branch publishing**: 5-8 minutes
- **Tagged release publishing**: 6-10 minutes

## Monitoring and Validation

### Post-Build Validation

After successful builds, the workflow validates:

1. **Image Accessibility**: Verifies images are accessible at registry
2. **Metadata Validation**: Checks OCI labels and metadata
3. **Multi-arch Support**: Confirms both architectures are available
4. **Registry Integration**: Tests registry API responses

### Success Reporting

The workflow provides comprehensive success reports:

- **Published image locations**: Direct links to registry
- **Pull commands**: Ready-to-use Docker commands
- **Metadata summary**: Build information and labels
- **Verification results**: Validation status and metrics

### Monitoring Integration

The workflow integrates with GitHub's monitoring:

- **Workflow status**: Visible in repository Actions tab
- **Job summaries**: Detailed reports in workflow runs
- **Error annotations**: GitHub UI error highlighting
- **Status badges**: Can be used in README files

## Troubleshooting Guide

### Common Issues

**1. Authentication Failures**
```
Error: failed to login to ghcr.io
```
**Resolution:**
- Verify workflow permissions include `packages: write`
- Check repository package settings
- Ensure workflow runs on main repository, not fork

**2. Build Failures**
```
Error: failed to build docker image
```
**Resolution:**
- Check Dockerfile syntax and dependencies
- Verify build context and .dockerignore
- Test build locally: `docker build --platform linux/amd64,linux/arm64 .`

**3. Multi-Architecture Issues**
```
Error: QEMU emulation setup failed
```
**Resolution:**
- Check GitHub runner availability
- Verify platform specifications
- Try building single architecture first

**4. Registry Access Issues**
```
Error: failed to push to registry
```
**Resolution:**
- Check GitHub Container Registry status
- Verify network connectivity
- Ensure registry permissions are correct

### Debugging Steps

1. **Check Workflow Logs**: Review detailed logs in GitHub Actions
2. **Validate Locally**: Test Docker build on local machine
3. **Check Dependencies**: Verify all dependencies are accessible
4. **Test Registry**: Manually test registry authentication
5. **Review Permissions**: Confirm all required permissions are set

### Getting Help

- **GitHub Issues**: Report problems in repository issues
- **GitHub Discussions**: Ask questions in repository discussions
- **Documentation**: Check this guide and inline workflow comments
- **Status Pages**: Monitor GitHub and registry service status

## Integration with Existing Tools

### GoReleaser Coordination

The workflow coordinates with GoReleaser to prevent conflicts:

- **Shared configuration**: Uses same Dockerfile and build arguments
- **Tag coordination**: Handles duplicate tags gracefully
- **Registry deduplication**: Registry manages overlapping publishes

See [GitHub Actions and GoReleaser Coordination](github-actions-goreleaser-coordination.md) for details.

### Makefile Compatibility

The workflow maintains compatibility with existing Makefile commands:

- **Build flags**: Uses same ldflags and build arguments
- **Docker targets**: Compatible with `make docker-build`
- **Version handling**: Consistent version extraction logic

### CI/CD Integration

The workflow integrates with other CI/CD tools:

- **Status checks**: Can be required for pull request merges
- **Deployment triggers**: Published images can trigger deployments
- **Notification hooks**: Supports webhook notifications

## Advanced Configuration

### Customizing Build Targets

To modify supported architectures, update the workflow:

```yaml
platforms: linux/amd64,linux/arm64,linux/arm/v7
```

### Custom Tagging

To add custom tags, modify the metadata extraction:

```yaml
tags: |
  type=ref,event=tag
  type=raw,value=latest,enable={{is_default_branch}}
  type=raw,value=stable,enable={{is_default_branch}}
```

### Environment-Specific Builds

For different environments, create separate workflows:

```yaml
# .github/workflows/docker-staging.yml
env:
  REGISTRY: ghcr.io
  IMAGE_NAME: powa-team/powa-sentinel-staging
```

### Custom Build Arguments

To add custom build arguments:

```yaml
build-args: |
  VERSION=${{ steps.build-meta.outputs.version }}
  COMMIT=${{ steps.build-meta.outputs.commit }}
  BUILD_DATE=${{ steps.build-meta.outputs.build_date }}
  ENVIRONMENT=production
  FEATURE_FLAGS=auth,metrics
```

## Maintenance

### Regular Maintenance Tasks

1. **Update Actions**: Keep GitHub Actions up to date
2. **Review Logs**: Monitor workflow execution logs
3. **Cache Management**: Monitor cache usage and effectiveness
4. **Security Updates**: Update base images and dependencies

### Workflow Updates

When updating the workflow:

1. **Test Changes**: Use pull requests to test modifications
2. **Validate Locally**: Test with `act` or similar tools
3. **Monitor Metrics**: Check performance impact of changes
4. **Document Changes**: Update this documentation

### Version Management

- **Workflow versioning**: Tag workflow changes for rollback
- **Action versions**: Pin action versions for stability
- **Dependency updates**: Regular updates with testing

## Best Practices

### Development Workflow

1. **Use Pull Requests**: Always test changes via pull requests
2. **Monitor Builds**: Check build status and logs regularly
3. **Test Locally**: Validate Docker builds before pushing
4. **Review Images**: Verify published images work correctly

### Security Practices

1. **Minimal Permissions**: Use least privilege principle
2. **Regular Updates**: Keep actions and dependencies updated
3. **Monitor Access**: Review repository and package permissions
4. **Audit Logs**: Regularly review workflow execution logs

### Performance Practices

1. **Optimize Dockerfile**: Use multi-stage builds and layer caching
2. **Monitor Cache**: Track cache hit rates and effectiveness
3. **Minimize Context**: Use .dockerignore to reduce build context
4. **Parallel Builds**: Leverage multi-architecture parallel building

## Conclusion

This GitHub Actions Docker publishing workflow provides a robust, secure, and efficient solution for automated Docker image publishing. It integrates seamlessly with existing development workflows while providing comprehensive error handling, monitoring, and validation capabilities.

For additional support or questions, please refer to the repository documentation or create an issue in the project repository.