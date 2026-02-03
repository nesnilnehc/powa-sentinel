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

## PoWA 仓库相关日志告警

若出现以下告警：

- **「pg_stat_kcache extension present but no history table found in PoWA 4」**  
  未找到 kcache 历史表。请确认安装 `pg_stat_kcache` 后已执行 `SELECT powa_kcache_register();`，且 powa-archivist 已创建对应表（通常在 `powa` schema）。Sentinel 会在 `public` 与 `powa` schema 下查找形如 `powa_%kcache%history` 的表。

- **「powa_qualstats_indexes view does not exist, skipping index suggestions」**  
  qualstats 未完整接入。请在 PoWA 数据库上以超级用户执行 `SELECT powa_qualstats_register();`，以便 PoWA 创建所需视图。若为自定义 PoWA 部署，请确保该视图存在且只读用户具备 `SELECT` 权限。

上述告警不会中断分析，慢查询与回归检测仍会执行，仅会跳过基于 kcache 的增强与索引建议。

### 「undefined_table」或「undefined_column」错误

若 PoWA 仓库数据库（即 powa-sentinel 所连接的数据库）返回未定义表或未定义列错误（例如在查询 `powa_statements_history` 或其他 PoWA 对象时），请确认你的 **PoWA（powa-archivist）版本**在[兼容性说明](../reference/compatibility.md)的支持矩阵内。不支持的版本或版本不匹配会使用不同 schema 并导致此类错误。请确保连接使用的 `search_path` 包含 PoWA 创建视图所在的 schema（通常为 `powa` 或 `public`）。
