# Configuration

powa-sentinel uses a YAML config file. All settings support environment variable interpolation (e.g. `${DB_HOST:-127.0.0.1}`).

## Quick reference

| Section | Key | Default | Description |
|---------|-----|---------|-------------|
| `database` | `host`, `port`, `user`, `password`, `dbname`, `sslmode` | — | PoWA repository connection |
| `schedule` | `cron` | `0 0 9 * * 1` | Cron expression (default: Monday 9:00) |
| `analysis` | `window_duration`, `comparison_offset` | `24h`, `168h` | Current vs baseline time windows |
| `rules` | `slow_sql`, `regression`, `index_suggestion` | — | Alert thresholds |
| `notifier` | `type`, `webhook_url`, `retries` | `console` | Notification channel |
| `server` | `port`, `deep_check` | `8080`, `true` | Health check endpoint |

## Example with env vars

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

## Notifier types

- **`console`**: Logs to stdout. Use for testing.
- **`wecom`**: Sends to WeCom webhook. Requires `webhook_url`.

For full field reference, see [Config Specification](../reference/config-spec.md). For deployment options, see [Deployment](../guides/deployment.md).
