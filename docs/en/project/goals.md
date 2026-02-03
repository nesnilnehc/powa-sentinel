# Project Goals

## Background

PoWA has complete performance collection and analysis, but only provides web visualization. It lacks active notifications, anomaly detection, and trend alerting. In practice, people do not actively check monitoring dashboards.

## Goal

Build an **independent, low-risk, long-running** PoWA push service that converts PoWA analysis results into actionable information and delivers them through enterprise channels (e.g. WeCom).

## Core Capabilities

1. **Data acquisition**: Periodically fetch performance stats from PoWA
2. **Scenario recognition**:
   - Slow SQL Top N (duration, or CPU/IO via `pg_stat_kcache`)
   - Abnormal growth in duration/count
   - Performance regression
   - Missing index suggestions (via `pg_qualstats` + hypopg)
3. **Message push**: WeCom (extensible to Feishu, DingTalk, Webhook)
4. **Non-functional**: Low intrusiveness, read-only, stability, maintainability, health checks

## Design Principles

- Push **conclusions**, not raw data
- Information grading: L1 (summary) → L2 (tech leads) → L3 (DBA details)
- Independent process, no PoWA code changes
- Read-only database access only

## Roadmap

1. Configurable rules (YAML / DSL)
2. Multi-instance / multi-tenancy
3. Push de-duplication and suppression
4. Correlation with release/change events
