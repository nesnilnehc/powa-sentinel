# Prerequisites

Before deploying powa-sentinel, ensure the following are in place.

## PoWA Installation

powa-sentinel connects to a **PoWA repository database**—the PostgreSQL instance where [PoWA](https://github.com/powa-team/powa) stores its history.

- **powa-archivist** must be installed and collecting data (schema `powa`).
- **Historical data** must be enabled and retained long enough for your analysis windows.
  - Example: weekly comparison needs at least 8 days of retention (`> 7 days`).

## Database User

Create a **read-only** user with access to the `powa` schema:

```sql
CREATE USER powa_readonly WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE powa TO powa_readonly;
GRANT USAGE ON SCHEMA powa TO powa_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA powa TO powa_readonly;
```

Never grant write or DDL privileges. powa-sentinel only reads aggregated views.

## Optional Extensions

| Extension | Purpose |
|-----------|---------|
| `pg_stat_kcache` | CPU/IO-based slow query detection |
| `pg_qualstats` | Missing index suggestions |

Install these on the PoWA repository database if you want richer alerts. After installing each extension, **register it with PoWA** (as superuser on the PoWA database):

```sql
SELECT powa_kcache_register();   -- for pg_stat_kcache
SELECT powa_qualstats_register(); -- for pg_qualstats
```

Without registration, the archivist will not create the history tables/views and you will see warnings (kcache enrichment disabled, index suggestions skipped).

## Environment expectation check (optional)

If you want to be warned when an extension you expect is not available (e.g. you intend to use kcache/qualstats but forgot to install or register), set `database.expected_extensions` in your config to a list such as `[pg_stat_kcache, pg_qualstats]`. On the first extension check, Sentinel will compare this list with what is actually available and log a message like: `Environment check: expected extensions [pg_stat_kcache pg_qualstats]; missing: [pg_qualstats]`. Leave the option unset or empty to skip this check. See [Config Specification](../reference/config-spec.md#database).

## Notification Credentials

- **WeCom (WeChat Work)**: Webhook URL from your WeCom group or app.
- **Other channels**: Not yet supported; use `console` for testing.

## Summary

| Requirement | Status |
|-------------|--------|
| PoWA with historical data | Required |
| Read-only DB user | Required |
| `pg_stat_kcache` | Optional |
| `pg_qualstats` | Optional |
| WeCom webhook | Required for production pushes |

Next: [Configuration](configuration.md)
