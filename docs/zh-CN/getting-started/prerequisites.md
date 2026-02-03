# 前置条件

部署 powa-sentinel 前请确保满足以下条件。

## PoWA 安装

powa-sentinel 连接 **PoWA 仓库数据库**——即 [PoWA](https://github.com/powa-team/powa) 存储历史数据的 PostgreSQL 实例。下文中也简称「仓库库」或「PoWA 库」。其与被监控的 PostgreSQL 实例在单机/多机下的关系见 [术语与部署模式](../reference/powa-schema.md#terminology)。

- **powa-archivist** 必须已安装并持续采集数据（schema `powa`）。
- **历史数据**必须启用，且保留时长满足分析窗口需求。
  - 例如：周同比分析至少需要 8 天（> 7 天）数据保留。

## 必须的扩展

以下扩展为 PoWA 与 powa-sentinel 运行所必需。

- **被监控的 PostgreSQL 实例**：在这些实例上安装并启用 `pg_stat_statements`。单机部署时被监控实例即仓库库；多机部署时在每台被监控实例上安装。PoWA 依赖其采集查询执行统计；未安装则没有 `powa_statements` 及历史数据（无法进行慢查询与回归分析）。
  - 示例：在**仓库库**（单机）或**每台被监控实例**（多机）上以超级用户执行：`CREATE EXTENSION pg_stat_statements;`
- **PoWA 仓库数据库**：必须存在 `powa` 扩展（由 powa-archivist 安装）。Sentinel 通过 `pg_extension.extversion` 读取 `powa` 版本以判断 3.x/4.x 并选择兼容的查询路径。（单机：与上为同一实例；多机：仅中心仓库库。）

完整组件与可选扩展说明见 [PoWA Schema 参考](../reference/powa-schema.md)。

## 数据库用户

创建具备 `powa` schema 访问权限的**只读**用户：

```sql
CREATE USER powa_readonly WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE powa TO powa_readonly;
GRANT USAGE ON SCHEMA powa TO powa_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA powa TO powa_readonly;
```

切勿授予写权限或 DDL。powa-sentinel 仅读取聚合视图。

## 可选扩展

| 扩展 | 用途 |
|------|------|
| `pg_stat_kcache` | 基于 CPU/IO 的慢查询检测 |
| `pg_qualstats` | 缺失索引建议 |

如需更丰富的告警，可在 **PoWA 仓库数据库** 上安装上述扩展。在**仓库库**上以超级用户**注册**。单机时仓库库即被监控实例；多机时仅中心仓库库有 `powa` schema 并需注册。

```sql
SELECT powa_kcache_register();   -- pg_stat_kcache
SELECT powa_qualstats_register(); -- pg_qualstats
```

未注册时，archivist 不会创建对应历史表/视图，会出现“禁用 kcache 增强”“跳过索引建议”等告警。

## 环境期望校验（可选）

若希望在某扩展未就绪时得到提示（例如打算使用 kcache/qualstats 但未安装或未注册），可在配置中设置 `database.expected_extensions`，例如 `[pg_stat_kcache, pg_qualstats]`。首次扩展检查时 Sentinel 会与当前实际可用扩展对比，并打出类似日志：`Environment check: expected extensions [pg_stat_kcache pg_qualstats]; missing: [pg_qualstats]`。不配置或留空则不进行对比。参见 [配置规范](../reference/config-spec.md#database)。

## 通知凭证

- **企业微信**：从企业微信群或应用获取 Webhook URL。
- **其他渠道**：暂不支持；测试时使用 `console`。

## 汇总

| 要求 | 状态 |
|------|------|
| PoWA 及历史数据 | 必需 |
| `pg_stat_statements` | 必需 |
| `powa` 扩展 | 必需 |
| 只读 DB 用户 | 必需 |
| `pg_stat_kcache` | 可选 |
| `pg_qualstats` | 可选 |
| 企业微信 webhook | 生产推送必需 |

下一步：[配置](configuration.md)
