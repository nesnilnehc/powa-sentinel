# powa-sentinel

[中文文档](./README_zh-CN.md)

powa-sentinel is a lightweight, read-only sidecar service for
[PoWA (PostgreSQL Workload Analyzer)](https://github.com/powa-team/powa).

It periodically analyzes PoWA performance statistics and proactively
pushes **actionable performance insights** to external notification
channels such as **WeCom (WeChat Work)**.

The project is designed for production environments where:

- PoWA is already deployed
- Teams want **proactive awareness** instead of manual dashboard checks
- Safety, stability, and low operational risk are critical

---

## Key Features

- 📊 Analyze PoWA aggregated query statistics
- ⏱ Detect slow queries, regressions, and abnormal workload changes
- 💡 **Proactive Index Suggestions** (via passive analysis of `pg_qualstats`)
- 🖥 **Resource-based Alerts** (CPU/IO) if `pg_stat_kcache` is integrated
- 📨 Push insights to WeCom (extensible to other channels)
- 🔐 Read-only access to PoWA data
- 🧩 Non-intrusive and decoupled from PoWA core

---

## Design Principles

- **Read-only by default**  
  The service never modifies database state or executes control actions.

- **Sidecar architecture**  
  powa-sentinel runs independently and does not affect PoWA availability.

- **Insight-driven notifications**  
  Notifications focus on trends and conclusions, not raw data dumps.

- **Production-friendly**  
  Single binary deployment, minimal dependencies, easy to operate.

---

## Architecture Overview

```
PostgreSQL
|
|  pg_stat_statements
v
PoWA (aggregation & analysis)
|
|  read-only access
v
powa-sentinel
|
v
Notification Channels (WeCom, Webhook, Email…)
```

## Documentation

See [docs/README.md](./docs/README.md) for the full index.

**Quick links:** [Quick Start](./docs/en/getting-started/quickstart.md) | [Deployment](./docs/en/guides/deployment.md) | [Contributing](./docs/en/guides/contributing.md)

## Quick Start

> ⚠️ Requires an existing PoWA installation. See [Prerequisites](./docs/en/getting-started/prerequisites.md).

```bash
# Create config.yaml (see docs), then:
docker run -d --restart unless-stopped -p 8080:8080 -v $(pwd)/config.yaml:/config.yaml ghcr.io/nesnilnehc/powa-sentinel:latest
curl http://localhost:8080/healthz
```

---

## Status

This project is in early development.
APIs, configuration formats, and rules are subject to change.

Contributions and discussions are welcome.

---

## License

This project is licensed under the PostgreSQL License.
See the [LICENSE](./LICENSE) file for details.
