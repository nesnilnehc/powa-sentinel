# 配置规范

规范配置模板见 [config/config.yaml.example](../../../config/config.yaml.example)。所有键支持 `${VAR:-default}` 形式的环境变量替换。

## 配置节

### database

| 键 | 类型 | 默认值 | 说明 |
|----|------|--------|------|
| `host` | string | `127.0.0.1` | PoWA 仓库数据库的主机（即 powa-sentinel 所连接的实例）。 |
| `port` | int | `5432` | 端口 |
| `user` | string | `powa_readonly` | 数据库用户 |
| `password` | string | — | 必填 |
| `dbname` | string | `powa` | 数据库名 |
| `sslmode` | string | `disable` | SSL 模式 |
| `expected_extensions` | string 列表 | *（空）* | 可选。期望可用的扩展（`pg_stat_kcache`、`pg_qualstats`）。若有缺失，扩展检查时会打出一条告警日志（环境期望校验）。不配置或留空则不进行对比。 |

### schedule

| 键 | 类型 | 默认值 | 说明 |
|----|------|--------|------|
| `cron` | string | `0 0 9 * * 1` | Cron 表达式（秒 分 时 日 月 周） |
| `timezone` | string | `UTC` | IANA 时区；cron 时间按此时区解析（如 `Asia/Shanghai`） |

### analysis

| 键 | 类型 | 默认值 | 说明 |
|----|------|--------|------|
| `window_duration` | duration | `24h` | 当前指标窗口 |
| `comparison_offset` | duration | `168h` | 基线偏移（如 7 天） |

### rules

| 键 | 子键 | 默认值 | 说明 |
|----|------|--------|------|
| `slow_sql` | `top_n` | `10` | 慢查询 Top N |
| `slow_sql` | `rank_by` | `total_time` | 指标：`total_time`、`mean_time`、`cpu_time`、`io_time` |
| `regression` | `threshold_percent` | `50` | 触发告警的最小涨幅 % |
| `index_suggestion` | `min_improvement_percent` | `30` | 纳入建议的最小预估收益 % |

### notifier

| 键 | 类型 | 默认值 | 说明 |
|----|------|--------|------|
| `type` | string | `console` | `console` 或 `wecom` |
| `webhook_url` | string | — | `type: wecom` 时必填 |
| `retries` | int | `3` | 重试次数 |
| `retry_delay` | duration | `1s` | 初始重试间隔（指数退避） |

### server

| 键 | 类型 | 默认值 | 说明 |
|----|------|--------|------|
| `port` | int | `8080` | 健康检查端口 |
| `deep_check` | bool | `true` | 健康检查是否包含 DB 连通性 |
