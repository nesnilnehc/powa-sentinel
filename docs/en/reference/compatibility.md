# PoWA Version Compatibility

powa-sentinel supports specific PoWA (powa-archivist) versions. The version is read from the repository database as `extversion` of the `powa` extension.

## Supported PoWA versions

| PoWA version | Support level | Notes |
|--------------|----------------|--------|
| **3.x** (3.0.0–3.2.0) | Supported | Single-server; flat history tables (`ts`, `total_time`/`total_exec_time`, `calls`); no `powa_servers`. Kcache history table: `powa_kcache_metrics_history`. |
| **4.x** (4.0.0–4.2.2) | Supported | Remote mode; `powa_servers`; history uses `records` array and `coalesce_range`. Kcache table name discovered (pattern `powa_%kcache%history`) in `public` and `powa` schemas. Run `powa_kcache_register()` and `powa_qualstats_register()` for optional features. |
| **5.x** | Best-effort | Treated like 4.x in code. May work if schema matches 4.x. PoWA 5 allows extensions in any schema; if objects are not in `public` or `powa`, table/view discovery may fail. |
| **1.x**, **2.x** | Not supported | Different schema and upgrade story; not tested or documented. |

## Per-version notes

- **3.x**: Single instance only. `powa_statements_history` has flat columns; no `srvid` or `powa_servers`. Sentinel uses the “PoWA 3” query path.
- **4+**: Multi-server; `srvid`, `powa_servers`, and `records`/`coalesce_range` in history. Sentinel uses the “PoWA 4” query path. Optional extensions require registration so that archivist creates the expected tables/views (e.g. in `powa` schema).
- **5.x**: No official statement that 5.x schema is identical to 4.x. If you use PoWA 5 and see errors, verify your PoWA version is in the support matrix and check the [troubleshooting guide](../operations/troubleshooting.md#powa-repository-log-warnings).

## See also

- [PoWA Schema Reference](powa-schema.md) — tables and views used by Sentinel
- [Prerequisites](../getting-started/prerequisites.md) — required setup and optional extensions
- [Troubleshooting](../operations/troubleshooting.md#powa-repository-log-warnings) — log warnings and version-related errors
