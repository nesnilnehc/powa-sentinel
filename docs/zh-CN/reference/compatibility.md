# PoWA 版本兼容性

powa-sentinel 仅支持特定的 PoWA（powa-archivist）版本。版本从仓库数据库中 `powa` 扩展的 `extversion` 读取。

## 支持的 PoWA 版本

| PoWA 版本 | 支持级别 | 说明 |
|-----------|----------|------|
| **3.x**（3.0.0–3.2.0） | 支持 | 单机；扁平 history 表（`ts`、`total_time`/`total_exec_time`、`calls`）；无 `powa_servers`。kcache 历史表：`powa_kcache_metrics_history`。 |
| **4.x**（4.0.0–4.2.2） | 支持 | 远程模式；`powa_servers`；history 使用 `records` 数组与 `coalesce_range`。kcache 表名通过发现（模式 `powa_%kcache%history`）在 `public` 与 `powa` schema 中查找。可选功能需执行 `powa_kcache_register()` 与 `powa_qualstats_register()`。 |
| **5.x** | 最佳-effort | 代码中按 4.x 处理。若 schema 与 4.x 一致可能可用。PoWA 5 允许扩展安装到任意 schema；若对象不在 `public` 或 `powa`，表/视图发现可能失败。 |
| **1.x**、**2.x** | 不支持 | 与当前使用的 3/4 两套 schema 不同；未测试且未在文档中承诺支持。 |

## 各版本说明

- **3.x**：仅单实例。`powa_statements_history` 为扁平列；无 `srvid` 或 `powa_servers`。Sentinel 使用「PoWA 3」查询路径。
- **4+**：多机；history 中有 `srvid`、`powa_servers` 及 `records`/`coalesce_range`。Sentinel 使用「PoWA 4」查询路径。可选扩展需注册后 archivist 才会创建对应表/视图（如在 `powa` schema）。
- **5.x**：官方未声明 5.x 与 4.x schema 完全一致。若使用 PoWA 5 并出现错误，请确认版本在支持矩阵内并参阅[故障排查](../operations/troubleshooting.md#powa-仓库相关日志告警)。

## 相关文档

- [PoWA Schema 参考](powa-schema.md) — Sentinel 使用的表与视图
- [前置条件](../getting-started/prerequisites.md) — 必需环境与可选扩展
- [故障排查](../operations/troubleshooting.md#powa-仓库相关日志告警) — 日志告警与版本相关错误
