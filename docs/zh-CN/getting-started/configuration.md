# 配置

powa-sentinel 使用 YAML 配置文件。所有项均支持环境变量插值（如 `${DB_HOST:-127.0.0.1}`）。

## 快速参考

| 节 | 键 | 默认值 | 说明 |
|----|-----|--------|------|
| `database` | `host`, `port`, `user`, `password`, `dbname`, `sslmode` | — | PoWA 仓库连接 |
| `schedule` | `cron`, `timezone` | `0 0 9 * * 1`, `UTC` | Cron 表达式；cron 时间按 `timezone`（IANA）解析 |
| `analysis` | `window_duration`, `comparison_offset` | `24h`, `168h` | 当前与基线时间窗口 |
| `rules` | `slow_sql`, `regression`, `index_suggestion` | — | 告警阈值 |
| `notifier` | `type`, `webhook_url`, `retries` | `console` | 通知渠道 |
| `server` | `port`, `deep_check` | `8080`, `true` | 健康检查端口 |

## 带环境变量示例

```yaml
database:
  host: "${DB_HOST:-127.0.0.1}"
  port: ${DB_PORT:-5432}
  user: "${DB_USER:-powa_readonly}"
  password: "${DB_PASSWORD}"
  dbname: "${DB_NAME:-powa}"
  sslmode: "${DB_SSLMODE:-disable}"

schedule:
  cron: "${SCHEDULE_CRON:-0 0 9 * * 1}"
  timezone: "${SCHEDULE_TZ:-UTC}"

analysis:
  window_duration: "${ANALYSIS_WINDOW:-24h}"
  comparison_offset: "${ANALYSIS_OFFSET:-168h}"

rules:
  slow_sql:
    top_n: ${RULES_SLOW_SQL_TOP_N:-10}
    rank_by: "${RULES_SLOW_SQL_RANK_BY:-total_time}"
  regression:
    threshold_percent: ${RULES_REGRESSION_THRESHOLD:-50}
  index_suggestion:
    min_improvement_percent: ${RULES_INDEX_MIN_IMPROVEMENT:-30}

notifier:
  type: "${NOTIFIER_TYPE:-console}"
  webhook_url: "${WECOM_WEBHOOK_URL}"
  retries: ${NOTIFIER_RETRIES:-3}

server:
  port: ${SERVER_PORT:-8080}
  deep_check: ${SERVER_DEEP_CHECK:-true}
```

## 通知类型

- **`console`**：输出到 stdout，用于测试。
- **`wecom`**：发送到企业微信 webhook，需设置 `webhook_url`。

完整字段说明见 [配置规范](../reference/config-spec.md)。部署方式见 [部署](../guides/deployment.md)。

## 调度与时区

Cron 表达式中的时间按配置项 **`schedule.timezone`** 解析，与容器或系统环境变量无关。默认为 `UTC`。使用 IANA 时区名（如 `Asia/Shanghai`、`Europe/London`）即可按本地时间执行：

```yaml
schedule:
  cron: "0 25 16 * * *"   # 每天 16:25 执行（见下方 timezone）
  timezone: "Asia/Shanghai"
```

启动日志会输出 `Scheduler started with cron: ... (timezone: ...)` 便于核对。
