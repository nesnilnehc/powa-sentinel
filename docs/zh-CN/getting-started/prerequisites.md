# 前置条件

部署 powa-sentinel 前请确保满足以下条件。

## PoWA 安装

powa-sentinel 连接 **PoWA 仓库数据库**——即 [PoWA](https://github.com/powa-team/powa) 存储历史数据的 PostgreSQL 实例。

- **powa-archivist** 必须已安装并持续采集数据（schema `powa`）。
- **历史数据**必须启用，且保留时长满足分析窗口需求。
  - 例如：周同比分析至少需要 8 天（> 7 天）数据保留。

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

如需更丰富的告警，可在 PoWA 仓库数据库上安装上述扩展。

## 通知凭证

- **企业微信**：从企业微信群或应用获取 Webhook URL。
- **其他渠道**：暂不支持；测试时使用 `console`。

## 汇总

| 要求 | 状态 |
|------|------|
| PoWA 及历史数据 | 必需 |
| 只读 DB 用户 | 必需 |
| `pg_stat_kcache` | 可选 |
| `pg_qualstats` | 可选 |
| 企业微信 webhook | 生产推送必需 |

下一步：[配置](configuration.md)
