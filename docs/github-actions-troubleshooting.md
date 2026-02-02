# GitHub Actions Docker Workflow Troubleshooting Guide

## Quick Diagnosis

Use this flowchart to quickly identify and resolve common issues:

```
Workflow Failed?
├── Authentication Error → Check Permissions Section
├── Build Error → Check Build Issues Section  
├── Registry Error → Check Registry Issues Section
├── Multi-arch Error → Check Platform Issues Section
└── Timeout Error → Check Performance Issues Section
```

## Common Error Messages and Solutions

### Authentication Errors

#### Error: "failed to login to ghcr.io"

**Symptoms:**
```
Error: failed to login to ghcr.io
Error: buildx failed with: error: failed to solve: failed to authorize: failed to fetch anonymous token
```

**Causes:**
- Insufficient GITHUB_TOKEN permissions
- Repository package settings misconfigured
- Workflow running on unauthorized fork

**Solutions:**

1. **Check Workflow Permissions**
   ```yaml
   permissions:
     contents: read
     packages: write  # ← This is required
     id-token: write
   ```

2. **Verify Repository Settings**
   - Go to Settings → Actions → General
   - Set "Workflow permissions" to "Read and write permissions"
   - Enable "Allow GitHub Actions to create and approve pull requests"

3. **Check Package Settings**
   - Go to Settings → Code and automation → Packages
   - Ensure packages are enabled for the repository
   - Verify package visibility settings

4. **Fork Repository Check**
   - Ensure workflow runs on main repository: `powa-team/powa-sentinel`
   - Forks cannot publish to the main repository's registry

#### Error: "GITHUB_TOKEN not available"

**Symptoms:**
```
❌ CRITICAL ERROR: GITHUB_TOKEN not available
```

**Causes:**
- Workflow permissions not configured
- Token scope insufficient

**Solutions:**
1. Add permissions block to workflow file
2. Check repository settings for Actions permissions
3. Verify organization settings don't restrict token access

### Build Errors

#### Error: "failed to build docker image"

**Symptoms:**
```
Error: failed to build docker image
Error: buildx failed with: error: failed to solve: process "/bin/sh -c ..." didn't complete successfully
```

**Causes:**
- Dockerfile syntax errors
- Missing dependencies
- Network connectivity issues
- Build context problems

**Solutions:**

1. **Test Locally**
   ```bash
   # Test single architecture
   docker build --platform linux/amd64 .
   
   # Test multi-architecture
   docker buildx build --platform linux/amd64,linux/arm64 .
   ```

2. **Check Dockerfile**
   - Verify all RUN commands complete successfully
   - Ensure base images are accessible
   - Check for typos in package names

3. **Validate Build Context**
   - Review `.dockerignore` file
   - Ensure required files are not excluded
   - Check file permissions

4. **Debug Build Arguments**
   ```yaml
   build-args: |
     VERSION=${{ steps.build-meta.outputs.version }}
     COMMIT=${{ steps.build-meta.outputs.commit }}
     BUILD_DATE=${{ steps.build-meta.outputs.build_date }}
   ```

#### Error: "go.mod not found" or "go.sum not found"

**Symptoms:**
```
Error: go.mod not found
Error: failed to compute cache key
```

**Causes:**
- Go module files missing from repository
- Build context doesn't include Go files
- Wrong working directory

**Solutions:**
1. Ensure `go.mod` and `go.sum` are committed to repository
2. Check `.dockerignore` doesn't exclude Go files
3. Verify Dockerfile WORKDIR and COPY commands

### Registry Errors

#### Error: "failed to push to registry"

**Symptoms:**
```
Error: failed to push to registry
Error: failed to upload blob: failed to authorize
```

**Causes:**
- Registry authentication failure
- Network connectivity issues
- Registry service unavailable
- Image size limits exceeded

**Solutions:**

1. **Check Registry Status**
   - Visit [GitHub Status](https://status.github.com)
   - Check Container Registry service status

2. **Verify Authentication**
   ```bash
   # Test registry access locally
   echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
   ```

3. **Check Image Size**
   - Large images may timeout during push
   - Optimize Dockerfile to reduce image size
   - Use multi-stage builds

4. **Network Issues**
   - Retry the workflow (transient network issues)
   - Check for GitHub runner connectivity problems

#### Error: "registry accessibility validation failed"

**Symptoms:**
```
❌ CRITICAL ERROR: X out of Y images are not accessible
```

**Causes:**
- Registry propagation delay
- Authentication issues during verification
- Registry service degradation

**Solutions:**
1. **Wait and Retry**: Registry propagation can take 1-2 minutes
2. **Check Registry Directly**: Visit GitHub Packages page
3. **Manual Verification**:
   ```bash
   docker pull ghcr.io/powa-team/powa-sentinel:latest
   ```

### Multi-Architecture Errors

#### Error: "QEMU emulation setup failed"

**Symptoms:**
```
Error: QEMU emulation setup failed
Error: failed to create builder instance
```

**Causes:**
- GitHub runner limitations
- QEMU installation issues
- Platform conflicts

**Solutions:**

1. **Retry Workflow**: Transient runner issues are common
2. **Check Platform Specification**:
   ```yaml
   platforms: linux/amd64,linux/arm64  # Ensure correct format
   ```

3. **Fallback to Single Architecture**:
   ```yaml
   platforms: linux/amd64  # Temporary fallback
   ```

4. **Verify Runner Capabilities**:
   - Some runners may not support all emulation features
   - Check GitHub runner specifications

#### Error: "buildx builder not found"

**Symptoms:**
```
Error: buildx builder not found
Error: failed to inspect builder
```

**Causes:**
- Docker Buildx setup failure
- Builder configuration issues
- Docker daemon problems

**Solutions:**

1. **Check Buildx Setup**:
   ```yaml
   - name: Set up Docker Buildx
     uses: docker/setup-buildx-action@v3
     with:
       driver: docker-container
       install: true
   ```

2. **Verify Docker Daemon**: Ensure Docker service is running
3. **Reset Builder**: Clear and recreate builder instance

### Performance and Timeout Errors

#### Error: "workflow timeout"

**Symptoms:**
```
Error: The operation was canceled.
Error: workflow timeout after 20 minutes
```

**Causes:**
- Large image builds
- Network connectivity issues
- Cache misses causing full rebuilds
- Resource constraints

**Solutions:**

1. **Increase Timeout**:
   ```yaml
   timeout-minutes: 30  # Increase from default 20
   ```

2. **Optimize Dockerfile**:
   - Use multi-stage builds
   - Minimize layer count
   - Order commands by change frequency

3. **Check Cache Effectiveness**:
   - Review cache hit rates in workflow logs
   - Ensure cache keys are stable

4. **Reduce Build Context**:
   ```dockerignore
   # .dockerignore
   .git
   .github
   docs
   test
   *.md
   ```

#### Error: "cache miss" or slow builds

**Symptoms:**
- Builds taking longer than expected
- "Cache miss" messages in logs
- Full dependency downloads every time

**Causes:**
- Cache key instability
- Cache storage limits exceeded
- Cache corruption

**Solutions:**

1. **Verify Cache Configuration**:
   ```yaml
   - uses: actions/cache@v4
     with:
       path: |
         ~/.cache/go-build
         ~/go/pkg/mod
       key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
   ```

2. **Check go.sum Stability**: Ensure go.sum doesn't change unnecessarily
3. **Monitor Cache Size**: Large caches may be evicted
4. **Clear Cache**: Manually clear cache if corrupted

### Metadata and Validation Errors

#### Error: "metadata validation failed"

**Symptoms:**
```
❌ CRITICAL ERROR: X out of Y metadata labels are invalid or missing
```

**Causes:**
- Missing OCI labels in image
- Incorrect label values
- Metadata extraction failure

**Solutions:**

1. **Check Label Configuration**:
   ```yaml
   labels: |
     org.opencontainers.image.title=powa-sentinel
     org.opencontainers.image.description=PostgreSQL Workload Analyzer Sentinel
     org.opencontainers.image.vendor=powa-team
   ```

2. **Verify Build Arguments**: Ensure metadata is passed correctly
3. **Test Image Labels**:
   ```bash
   docker inspect ghcr.io/powa-team/powa-sentinel:latest | jq '.[0].Config.Labels'
   ```

#### Error: "version metadata generation failed"

**Symptoms:**
```
❌ CRITICAL ERROR: Failed to generate version metadata
```

**Causes:**
- Git context unavailable
- Invalid Git references
- Checkout issues

**Solutions:**
1. **Check Checkout Step**: Ensure repository is properly checked out
2. **Verify Git Context**: Check `github.ref` and `github.sha` availability
3. **Test Locally**: Verify Git tags and branches exist

## Debugging Workflow Execution

### Enable Debug Logging

Add these secrets to your repository for enhanced debugging:

```
ACTIONS_STEP_DEBUG = true
ACTIONS_RUNNER_DEBUG = true
```

### Workflow Debugging Steps

1. **Review Workflow Logs**
   - Go to Actions tab → Select failed workflow
   - Expand each step to see detailed logs
   - Look for error messages and stack traces

2. **Check Job Summary**
   - Each workflow run includes a detailed summary
   - Review validation results and metrics
   - Check error analysis sections

3. **Validate Inputs**
   - Verify trigger conditions are met
   - Check environment variables and secrets
   - Confirm repository settings

4. **Test Components Individually**
   - Test Docker build locally
   - Verify registry authentication
   - Check multi-architecture support

### Local Testing with Act

Use [act](https://github.com/nektos/act) to test workflows locally:

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Test workflow locally
act push

# Test specific job
act -j docker

# Test with specific event
act push -e .github/workflows/test-event.json
```

## Prevention Strategies

### Pre-commit Checks

1. **Dockerfile Linting**:
   ```bash
   # Install hadolint
   brew install hadolint
   
   # Lint Dockerfile
   hadolint Dockerfile
   ```

2. **Workflow Validation**:
   ```bash
   # Install actionlint
   brew install actionlint
   
   # Validate workflow files
   actionlint .github/workflows/*.yml
   ```

3. **Local Build Testing**:
   ```bash
   # Test multi-arch build
   docker buildx build --platform linux/amd64,linux/arm64 .
   
   # Test with build args
   docker build --build-arg VERSION=test .
   ```

### Monitoring and Alerts

1. **Workflow Status Badges**: Add status badges to README
2. **Notification Setup**: Configure workflow failure notifications
3. **Regular Reviews**: Periodically review workflow performance
4. **Dependency Updates**: Keep actions and dependencies updated

### Best Practices

1. **Gradual Changes**: Test workflow changes in pull requests
2. **Documentation**: Keep workflow documentation updated
3. **Version Pinning**: Pin action versions for stability
4. **Error Handling**: Include comprehensive error handling
5. **Monitoring**: Monitor workflow execution metrics

## Getting Additional Help

### Resources

- **GitHub Actions Documentation**: https://docs.github.com/en/actions
- **Docker Buildx Documentation**: https://docs.docker.com/buildx/
- **GitHub Container Registry**: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry

### Support Channels

1. **Repository Issues**: Create an issue in the project repository
2. **GitHub Community**: Ask questions in GitHub Community discussions
3. **Stack Overflow**: Search for similar issues with `github-actions` tag
4. **GitHub Support**: Contact GitHub Support for platform issues

### Escalation Process

1. **Level 1**: Check this troubleshooting guide
2. **Level 2**: Search existing issues and documentation
3. **Level 3**: Create detailed issue with logs and reproduction steps
4. **Level 4**: Contact maintainers or GitHub Support

## Workflow Health Checklist

Use this checklist to maintain workflow health:

- [ ] Workflow runs successfully on pull requests
- [ ] Images publish correctly for main branch
- [ ] Tagged releases create proper version tags
- [ ] Multi-architecture builds complete successfully
- [ ] Registry accessibility validation passes
- [ ] Metadata validation passes
- [ ] Cache hit rates are reasonable (>50%)
- [ ] Build times are within acceptable limits
- [ ] Error handling works correctly
- [ ] Documentation is up to date

Regular maintenance using this checklist helps prevent issues and ensures reliable workflow operation.