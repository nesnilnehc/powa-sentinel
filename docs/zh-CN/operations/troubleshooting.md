# Workflow 故障排查

诊断并修复 CI/CD 故障（main/PR 的 CI、tag 推送的 Release）。

**面向**：维护者。发布详情见 [发布流程](release-workflow.md)，部署见 [部署指南](../guides/deployment.md)。

## 快速诊断

```
Workflow 失败？
├── 认证错误 → 权限
├── 构建错误 → 构建问题
├── 镜像仓库错误 → Registry 问题
├── 多架构错误 → 平台问题
└── 超时错误 → 性能问题
```

## 认证

### "failed to login to ghcr.io"

- 确认 **设置 → Actions → 常规** → “读取和写入权限”
- 确认 workflow 具备 `packages: write`
- Fork 仓库无法向主仓库发布

### "Registry authentication failed after retry"

1. 将 workflow 权限设为读写
2. 可选：添加 `GHCR_TOKEN` 密钥（具备 `write:packages` 的 PAT）

## 构建

### "failed to build docker image"

- 本地测试：`docker build --platform linux/amd64 .`
- 检查 Dockerfile 和 `.dockerignore`
- 确认 `go.mod`、`go.sum` 已提交

## 镜像仓库

### "failed to push to registry"

- 查看 [GitHub Status](https://status.github.com)
- 测试认证：`echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin`

## 多架构

### "QEMU emulation setup failed"

- 重试（偶发）
- 或临时仅使用 `platforms: linux/amd64`

## 调试

- 添加密钥：`ACTIONS_STEP_DEBUG=true`、`ACTIONS_RUNNER_DEBUG=true`
- 本地测试：[act](https://github.com/nektos/act)
- 语法检查：`actionlint .github/workflows/*.yml`
