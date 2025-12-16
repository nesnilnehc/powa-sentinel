# powa-sentinel

[English](./README.md)

powa-sentinel 是一个轻量级、只读的 Sidecar 服务，专为 [PoWA (PostgreSQL Workload Analyzer)](https://github.com/powa-team/powa) 设计。

它定期分析 PoWA 性能统计数据，并将 **可操作的性能洞察** 主动推送到外部通知渠道，例如 **企业微信 (WeCom)**。

本项目专为生产环境设计，适用于以下场景：

- PoWA 已经部署
- 团队希望获得 **主动感知**，而不是手动检查仪表盘
- 安全性、稳定性和低运维风险至关重要

---

## 主要特性

- 📊 分析 PoWA 聚合查询统计信息
- ⏱ 检测慢查询、性能衰退和异常工作负载变化
- 💡 **主动索引建议**（通过被动分析 `pg_qualstats`）
- 🖥 **基于资源的告警**（CPU/IO，如果集成了 `pg_stat_kcache`）
- 📨 推送洞察到企业微信（可扩展至其他渠道）
- 🔐 对 PoWA 数据的只读访问
- 🧩 非侵入式，与 PoWA 核心解耦

---

## 设计原则

- **默认只读**  
  服务从不修改数据库状态或执行控制操作。

- **Sidecar 架构**  
  powa-sentinel 独立运行，不影响 PoWA 的可用性。

- **洞察驱动的通知**  
  通知关注趋势和结论，而不是原始数据转储。

- **生产友好**  
  单二进制文件部署，依赖极少，易于操作。

---

## 架构概览

```
PostgreSQL
|
|  pg_stat_statements
v
PoWA (聚合与分析)
|
|  只读访问
v
powa-sentinel
|
v
通知渠道 (企业微信, Webhook, Email…)
```

## 文档

- [产品需求](./docs/zh-CN/product/requirements.md)
- [架构与设计](./docs/zh-CN/dev/design.md)
- [PoWA 参考](./docs/zh-CN/dev/powa_reference.md)
- [部署指南](./docs/zh-CN/ops/deployment.md)

## 快速开始

> ⚠️ 本项目假设已存在 PoWA 安装环境。

### 要求

- 已安装 PoWA 的 PostgreSQL
- 具有访问 PoWA 模式权限的只读数据库用户
- **推荐扩展**（在 PoWA 仓库上）：
  - `pg_stat_kcache`: 用于基于 CPU/IO 的慢 SQL 检测。
  - `pg_qualstats`: 用于缺失索引建议。
- 企业微信 Webhook 或应用凭证

---

## 状态

本项目处于早期开发阶段。
API、配置格式和规则可能会发生变化。

欢迎贡献和讨论。

---

## 许可证

本项目采用 PostgreSQL 许可证。
详情请参阅 [LICENSE](./LICENSE) 文件。
