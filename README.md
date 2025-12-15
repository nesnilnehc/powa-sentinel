# powa-sentinel

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

## Getting Started

> ⚠️ This project assumes an existing PoWA installation.

### Requirements

- PostgreSQL with PoWA installed
- Read-only database user with access to PoWA schemas
- **Recommended Extensions** (on PoWA repository):
    - `pg_stat_kcache`: For CPU/IO-based slow SQL detection.
    - `pg_qualstats`: For missing index suggestions.
- WeCom webhook or application credentials

---

## Status

This project is in early development.
APIs, configuration formats, and rules are subject to change.

Contributions and discussions are welcome.

---

## License

This project is licensed under the PostgreSQL License.
See the [LICENSE](./LICENSE) file for details.
