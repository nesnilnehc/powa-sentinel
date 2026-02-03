# 技术架构

## 系统定位

- **角色**：PoWA 的 Sidecar / Sentinel
- **特点**：
  - 不修改 PoWA 代码
  - 仅消费 PoWA 已生成的数据

## 架构

```mermaid
graph TD
    PG[PostgreSQL] --> STAT[pg_stat_statements]
    STAT --> POWA[PoWA (Analyze / Aggregate)]
    POWA --> PUSH[Push Service (Read Only)]
    PUSH --> WECHAT[WeCom / Channels]
```

## 数据访问原则

- **账户**：独立只读账户
- **范围**：仅 `powa` schema 聚合视图
- **限制**：不连接业务库，不做 DDL/DML
- **参考**：见 [PoWA Schema](powa-schema.md)

## 项目布局

```
.
├── cmd/powa-sentinel/      # 入口
├── config/                 # 配置模板
├── internal/
│   ├── config/             # 配置实现
│   ├── model/              # 数据结构
│   ├── reader/             # 数据库交互
│   ├── engine/             # 逻辑引擎
│   ├── notifier/           # 推送实现
│   ├── server/             # HTTP 健康检查
│   └── scheduler/          # 任务调度
└── pkg/                    # 可复用包
```

配置说明见 [配置规范](config-spec.md)。

## 核心数据模型

### MetricSnapshot

```go
type MetricSnapshot struct {
    QueryID   int64
    Query     string
    TotalTime float64
    MeanTime  float64
    Calls     int64
}
```

### AlertContext

```go
type AlertContext struct {
    ReqID        string
    ReportType   string
    TopSlowSQL   []MetricSnapshot
    Regressions  []RegressionItem
    Suggestions  []IndexSuggestion
}
```

### IndexSuggestion

```go
type IndexSuggestion struct {
    Table                 string
    AccessType            string
    Columns               []string
    EstImprovementPercent float64
}
```

## 组件逻辑

### Reader

- **目标**：`powa_statements`（实时）、`powa_statements_history`（基线）、`powa_qualstats_indexes`
- **策略**：按时间窗口 `ts` 查询

### Rule Engine

1. **Top N**：按 `TotalTime` DESC 排序（或 `pg_stat_kcache` I/O）
2. **Regression**：`(Current.MeanTime - Baseline.MeanTime) / Baseline.MeanTime`
3. **Suggestion**：过滤 `powa_qualstats_indexes` 高收益（>30%）缺失索引

### Notifier

- **重试**：指数退避（1s、2s、4s）

### Health Server

- **端点**：`GET /healthz`
- **逻辑**：进程运行且能连接 DB 时返回 200 OK（可选深度检查）

## 执行流程

1. Scheduler 触发任务
2. Reader 拉取当前与基线数据
3. Engine 分析并生成 `AlertContext`
4. Notifier 格式化并发送
