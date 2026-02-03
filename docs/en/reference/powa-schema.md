# PoWA Schema Reference

Context on [PoWA](https://github.com/powa-team/powa) and the data schema used by powa-sentinel.

## Overview

PoWA is a PostgreSQL Workload Analyzer. powa-sentinel acts as a sidecar: it queries **aggregations** produced by PoWA, not raw PostgreSQL stats. This minimizes overhead and aligns with what DBAs see in the PoWA UI.

## PoWA Ecosystem

| Component | Role | Relation to powa-sentinel |
|-----------|------|---------------------------|
| **powa-archivist** | Background worker, stores data in `powa` schema | powa-sentinel connects to this DB |
| **powa-web** | Web UI for visualization | Parallel; powa-sentinel for automated alerting |
| **pg_stat_statements** | Query execution stats | Mandatory |
| **pg_stat_kcache** | CPU/IO metrics | Optional |
| **pg_qualstats** | Index suggestions | Optional |

## Key Concepts

- **Repository database**: PostgreSQL instance where PoWA stores history (schema `powa`)
- **Snapshot**: PoWA collects stats periodically (default every 5 minutes)
- **Aggregation**: Data aggregated by time windows

## Data Dictionary

### powa_statements (view)

| Field | Type | Description |
|-------|------|-------------|
| `queryid` | bigint | Normalized query identifier |
| `query` | text | Normalized query text |
| `calls` | bigint | Execution count |
| `total_time` | double | Total execution time (ms) |
| `mean_time` | double | Mean time per call (ms) |
| `ts` | timestamp | Snapshot/aggregation window time |

### powa_statements_history (table/view)

Historical aggregated stats for trends and regression.

| Field | Type | Usage |
|-------|------|-------|
| `queryid` | bigint | Join with powa_statements |
| `ts` / `coalesce_range` | timestamp/tstzrange | Time-series filtering |
| `calls` | bigint | Volume analysis |
| `total_time` | double | Performance analysis |
| `mean_time` | double | Regression calculation (baseline) |

### powa_databases

| Field | Type | Description |
|-------|------|-------------|
| `dbid` | oid | Database OID |
| `datname` | text | Database name |

## Supported Extensions

- **pg_stat_kcache**: CPU/IO-based slow query analysis
- **pg_qualstats**: Missing index suggestions (passive read)
- **pg_wait_sampling**: Future support for lock-related alerts

## See also

- [PoWA Version Compatibility](compatibility.md) — supported PoWA versions (3.x, 4.x; 5.x best-effort) and per-version notes
