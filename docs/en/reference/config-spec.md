# Config Specification

The canonical config template is [config/config.yaml.example](../../../config/config.yaml.example). All keys support `${VAR:-default}` style environment substitution.

## Sections

### database

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `127.0.0.1` | Host of the PoWA repository database (the instance powa-sentinel connects to). |
| `port` | int | `5432` | Port |
| `user` | string | `powa_readonly` | Database user |
| `password` | string | — | Required |
| `dbname` | string | `powa` | Database name |
| `sslmode` | string | `disable` | SSL mode |
| `expected_extensions` | list of string | *(empty)* | Optional. Extensions you expect to be available (`pg_stat_kcache`, `pg_qualstats`). If any are missing, a warning is logged at extension check (environment expectation check). Omit or leave empty to skip comparison. |

### schedule

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `cron` | string | `0 0 9 * * 1` | Cron expression (second minute hour day month dow) |
| `timezone` | string | `UTC` | IANA timezone; cron times are interpreted in this zone (e.g. `Asia/Shanghai`) |

### analysis

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `window_duration` | duration | `24h` | Current metrics window |
| `comparison_offset` | duration | `168h` | Baseline offset (e.g. 7 days) |

### rules

| Key | Sub-key | Default | Description |
|-----|---------|---------|-------------|
| `slow_sql` | `top_n` | `10` | Top N slow queries |
| `slow_sql` | `rank_by` | `total_time` | Metric: `total_time`, `mean_time`, `cpu_time`, `io_time` |
| `regression` | `threshold_percent` | `50` | Min % increase to alert |
| `index_suggestion` | `min_improvement_percent` | `30` | Min estimated gain to include |

### notifier

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | `console` | `console` or `wecom` |
| `webhook_url` | string | — | Required when `type: wecom` |
| `retries` | int | `3` | Retry attempts |
| `retry_delay` | duration | `1s` | Initial retry delay (exponential backoff) |

### server

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `port` | int | `8080` | Health check port |
| `deep_check` | bool | `true` | Include DB connectivity in health check |
