# 技术架构与设计

## 1. 系统定位与架构

### 1.1 定位

* **角色**: PoWA 的 Sidecar / 哨兵 (Sentinel)。
* **特性**:
  * 不修改 PoWA 代码。
  * 仅消费 PoWA 已经生成的数据。

### 1.2 架构关系

```mermaid
graph TD
    PG[PostgreSQL] --> STAT[pg_stat_statements]
    STAT --> POWA[PoWA (分析 / 聚合)]
    POWA --> PUSH[推送服务 (只读)]
    PUSH --> WECHAT[企业微信 / 渠道]
```

### 1.3 数据访问原则

* **账户**: 独立的只读账户。
* **范围**: 仅限 `powa` 模式的聚合视图。
* **限制**: 不连接业务数据库，无 DDL/DML。
* **参考**: 模式详情请参见 [PoWA 参考](./powa_reference.md)。

## 2. 项目布局

标准 Go 项目布局：

```
.
├── cmd/powa-sentinel/      # 入口点
├── config/                 # 配置模板
├── internal/
│   ├── config/             # 配置实现
│   ├── model/              # 数据结构
│   ├── reader/             # 数据库交互
│   ├── engine/             # 逻辑引擎
│   ├── notifier/           # 推送实现
│   ├── server/             # HTTP 服务器 (健康检查)
│   └── scheduler/          # 任务控制
└── pkg/                    # 可复用包
```

## 3. 配置规范

```yaml
database:
  host: "${DB_HOST:-127.0.0.1}" # 支持环境变量
  dbname: "powa"

schedule:
  cron: "0 9 * * 1" 

analysis:
  window_duration: "24h" 
  comparison_offset: "168h" 
  
rules:
  slow_sql: { top_n: 10 }
  regression: { threshold_percent: 50 }

notifier:
  type: "wecom"
  retries: 3
```

## 4. 核心数据模型

### 4.1 MetricSnapshot

```go
type MetricSnapshot struct {
    QueryID      int64
    Query        string
    TotalTime    float64
    MeanTime     float64
    Calls        int64
}
```

### 4.2 AlertContext

```go
type AlertContext struct {
    ReqID        string
    ReportType   string 
    TopSlowSQL   []MetricSnapshot
    Regressions  []RegressionItem
    Suggestions  []IndexSuggestion
}
```

### 4.3 IndexSuggestion

```go
type IndexSuggestion struct {
    Table      string
    AccessType string // 例如: Seq Scan
    Columns    []string
    EstImprovementPercent float64 // HypoPG 影响
}
```

## 5. 组件逻辑

### 5.1 Reader (读取器)
*   **目标**: 
    *   `powa_statements` (实时) & `powa_statements_history` (基线)。
    *   `powa_qualstats_indexes` (建议索引的只读视图)。
*   **策略**: 通过时间窗口 `ts` 范围查询。

### 5.2 Rule Engine (规则引擎)
1.  **Top N**: 按 `TotalTime` 降序排序。（如果启用了 `pg_stat_kcache`，则检查 I/O 时间）。
2.  **Regression (回归)**: `(当前.MeanTime - 基线.MeanTime) / 基线.MeanTime`。
3.  **Suggestion (建议)**: 筛选 `powa_qualstats_indexes` 中高影响 (>30%) 的缺失索引。

### 5.3 Notifier (通知器)
*   **重试策略**: 针对网络故障实施指数退避（例如 1s, 2s, 4s）。

### 5.4 Health Server (健康检查服务器) (`internal/server`)
*   **端点**: `GET /healthz`
*   **逻辑**: 如果进程正在运行且可以连接到数据库，则返回 200 OK（可选的深度检查）。

## 6. 执行流程
1.  **Scheduler (调度器)** 触发任务。
2.  **Reader (读取器)** 获取当前和基线数据。
3.  **Engine (引擎)** 分析并生成 `AlertContext`。
4.  **Notifier (通知器)** 格式化并发送载荷。
