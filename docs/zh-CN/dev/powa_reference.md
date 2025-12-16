# PoWA (PostgreSQL Workload Analyzer) 参考

本文档提供了上游 [PoWA](https://github.com/powa-team/powa) 项目的背景信息，并定义了 `powa-sentinel` 使用的数据模式。

## 1. 项目概览

**PoWA** 是一个 PostgreSQL 工作负载分析器，它收集性能统计数据并提供实时图表。它依赖 `pg_stat_statements` 扩展来收集查询性能数据。

`powa-sentinel` 充当 PoWA 的 Sidecar。它不是直接从 PostgreSQL 收集原始数据，而是查询 PoWA 已经处理和存储的 **聚合数据**。这确保了：
* 对数据库的开销最小。
* 与 DBA 在 PoWA UI 中看到的数据一致。

## 2. PoWA 生态系统

了解 PoWA 生态系统有助于阐明 `powa-sentinel` 的边界。

*   **`powa-archivist` (核心)**: 这是一个运行后台工作进程的 PostgreSQL 扩展。它负责：
    *   从 `pg_stat_statements` 创建数据快照。
    *   将数据存储在 `powa` 数据库/模式中。
    *   **关系**: `powa-sentinel` 连接到安装了 `powa-archivist` 的数据库。

*   **`powa-web` (UI)**: 一个基于 Web 的仪表盘，供人类可视化数据。
    *   **关系**: 与 `powa-sentinel` 平行。`powa-sentinel` 用于自动告警，而 `powa-web` 用于手动调查。告警最终可能会链接到 `powa-web`。

*   **数据源插件**:
    *   `pg_stat_statements`: (强制) 提供 SQL 查询执行统计信息（时间、调用次数）。
    *   `pg_stat_kcache`: (可选) 提供 CPU 和磁盘 I/O 指标。
    *   `pg_qualstats`: (可选) 提供索引和谓词分析。

## 3. 关键概念

*   **仓库数据库**: PoWA 存储其历史记录的 PostgreSQL 实例（模式 `powa`）。
*   **快照**: PoWA 定期收集统计数据（默认每 5 分钟）。
*   **聚合**: 数据按时间窗口聚合（例如，每个快照的查询统计信息）。

## 3. 数据字典

`powa-sentinel` 读取器组件主要与 `powa` 模式中的以下视图交互。

### 3.1 `powa_statements` (视图)

此视图提供聚合的查询统计信息。在 `powa-sentinel` 的上下文中，我们关注映射到内部 `MetricSnapshot` 模型的字段。

| 字段名 | 类型 | 描述 | 映射 |
| :--- | :--- | :--- | :--- |
| `queryid` | `bigint` | 规范化查询文本的唯一标识符。 | `MetricSnapshot.QueryID` |
| `query` | `text` | 规范化的查询文本（占位符代替值）。 | `MetricSnapshot.Query` |
| `calls` | `bigint` | 查询执行的次数。 | `MetricSnapshot.Calls` |
| `total_time` | `double` | 执行查询花费的总时间 (ms)。 | `MetricSnapshot.TotalTime` |
| `mean_time` | `double` | 每次调用的平均执行时间 (ms)。 | `MetricSnapshot.MeanTime` |
| `ts` | `timestamp` | 快照/聚合窗口的时间戳。 | 用于时间窗口过滤 |

> **注意**: PoWA 支持表（例如 `powa_statements_src`）中的实际列名可能因 PoWA 版本而略有不同，但 `powa-sentinel` 依赖于标准视图接口。

### 3.2 `powa_statements_history` (表/视图)

此表存储历史聚合性能统计信息。它对于检测随时间变化的趋势和回归至关重要。

| 字段名 | 类型 | 描述 | 用法 |
| :--- | :--- | :--- | :--- |
| `queryid` | `bigint` | 查询的唯一标识符。 | 与 `powa_statements` 关联 |
| `ts` 或 `coalesce_range` | `timestamp/tstzrange` | 聚合的时间窗口。 | 时间序列分析 |
| `calls` | `bigint` | 此时间窗口内的调用次数。 | 数量分析 |
| `total_time` | `double` | 此窗口内的总执行时间。 | 性能分析 |
| `mean_time` | `double` | 此窗口内每次调用的平均时间。 | 回归计算 |

> **关键**: **回归率** 逻辑依赖于将 *当前窗口* (实时数据) 的 `mean_time` 与此历史表 (基线数据) 中的 `mean_time` 进行比较。

### 3.3 `powa_databases` (表)

将内部 ID 映射到人类可读的数据库名称。

| 字段名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `dbid` | `oid` | 数据库 OID。 |
| `datname` | `text` | 数据库名称。 |

## 4. 衍生指标

## 5. 支持的扩展和数据源

`powa-sentinel` 支持以下扩展以增强告警。

### 5.1 核心范围

*   **`pg_stat_kcache`**:
    *   **价值**: 向查询统计信息添加物理资源指标（CPU、磁盘 I/O）。
    *   **用法**: 用于分析 "CPU/IO 密集型" 慢查询。

*   **`pg_qualstats`**:
    *   **价值**: 建议缺失的索引。
    *   **用法**: 识别 "优化机会"（索引建议）。
    *   **机制**: 被动读取 PoWA 生成的建议。

*   **`pg_wait_sampling`**:
    *   **价值**: 提供慢查询背后的 "原因"。
    *   **用法**: 未来支持 "锁定告警"。
