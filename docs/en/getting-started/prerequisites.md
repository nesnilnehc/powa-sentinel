# Prerequisites

Before deploying powa-sentinel, ensure the following are in place.

## PoWA Installation

powa-sentinel connects to a **PoWA repository database**—the PostgreSQL instance where [PoWA](https://github.com/powa-team/powa) stores its history. In this doc we also call it the *repository* or *repository database*. For how it relates to monitored PostgreSQL instances (single- vs multi-server), see [Terminology and deployment modes](../reference/powa-schema.md#terminology).

- **powa-archivist** must be installed and collecting data (schema `powa`).
- **Historical data** must be enabled and retained long enough for your analysis windows.
  - Example: weekly comparison needs at least 8 days of retention (`> 7 days`).

## Required extensions

The following extensions must be present for PoWA and powa-sentinel to work.

- **Monitored PostgreSQL instance(s)**: Install and enable `pg_stat_statements` on these instances. In single-server deployments the monitored instance is the same as the repository; in multi-server, install on each monitored instance. PoWA relies on it for query execution statistics; without it there is no `powa_statements` or history data (no slow-query or regression analysis).
  - Example: as superuser on the **repository database** (single-server) or on **each monitored instance** (multi-server): `CREATE EXTENSION pg_stat_statements;`
- **PoWA repository database**: The `powa` extension must exist (powa-archivist installs it). Sentinel reads `pg_extension.extversion` for `powa` to detect 3.x/4.x and choose the compatible query path. (Single-server: same host as above; multi-server: the central repository host only.)

For the full component list and optional extensions, see [PoWA Schema Reference](../reference/powa-schema.md).

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

Install these on the **PoWA repository database** if you want richer alerts. Register as superuser **on the repository database**. In single-server setups the repository is the same as the monitored instance; in multi-server, only the central repository has the `powa` schema and registration.

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
| `pg_stat_statements` | Required |
| `powa` extension | Required |
| Read-only DB user | Required |
| `pg_stat_kcache` | Optional |
| `pg_qualstats` | Optional |
| WeCom webhook | Required for production pushes |

Next: [Configuration](configuration.md)
