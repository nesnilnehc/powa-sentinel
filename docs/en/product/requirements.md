# Product Requirements

## 1. Background & Goals

### 1.1 Background

* **Current State**: PoWA (PostgreSQL Workload Analyzer) already possesses complete performance collection and analysis capabilities.
* **Pain Points**:
  * Currently, it only provides Web visualization.
  * It lacks active notifications, anomaly detection, and trend alerting capabilities.
  * In enterprise operations and R&D collaboration, "people do not actively check monitoring" is the norm.

### 1.2 Goals

Build an **independent, low-risk, and long-running** PoWA Push Service to achieve:
> Converting PoWA's "analysis results" into actionable information and actively reaching relevant personnel through enterprise communication channels (e.g., WeCom).

## 2. Core Requirements

### 2.1 Functional Requirements

The system must implement the following core functions:

1. **Data Acquisition**:
    * Periodically fetch performance statistics from the PoWA database.
2. **Scenario Recognition**:
    * Identify **Slow SQL Top N** based on rules:
        * By Duration (Standard).
        * By CPU / IO (via `pg_stat_kcache`, if available).
    * Identify **Abnormal growth in query duration/count**.
    * Identify **Performance Regression**.
    * Identify **Optimization Opportunities** (New):
        * **Missing Index Suggestions** (via `pg_qualstats` + `hypopg`).
3. **Message Push**:
    * Push analysis conclusions as messages to Enterprise WeChat (WeCom).
4. **Extensibility**:
    * Support future possibilities for Feishu, DingTalk, Email, or Webhook.

### 2.2 Non-Functional Requirements

1. **Low Intrusiveness**:
    * Does not affect PoWA's normal operation.
    * Does not introduce database control behaviors.
2. **Security**:
    * Does not access business data.
3. **Stability**:
    * Can run stably for a long time.
4. **Maintainability**:
    * Low deployment and operation costs.
5. **Self-Monitoring**:
    * The service must expose its own health status (Health Checks) for container orchestration systems.

### 2.3 Prerequisites

To ensure the system can correctly identify "trends" and "anomalies", the target PoWA instance must meet the following conditions:

1. **Historical Data Storage**: Historical data persistence must be enabled.
2. **Data Retention Period**: The data retention duration in PoWA must cover the required analysis window.
    * e.g., If "Weekly YoY" analysis is required, PoWA needs to retain data for at least 8 days (`> 7 days`).

## 3. Push Content Design Principles

### 3.1 Core Principles

Push **"Conclusions"**, not "Raw Data". Message content should focus on: Metrics, Comparisons, Trends, Impact Assessment.

### 3.2 Information Grading

* **L1 (Management)**: Summary conclusions (e.g., "Database health dropped to 80%").
* **L2 (Tech Leads)**:
  * **Performance**: Slow SQL HASH + Duration/CPU change.
  * **Optimization**: "Table X is missing index for query Y (Est. gain: 50%)".
* **L3 (DBA)**: Full SQL text, Execution Plan link, direct `powa-web` URL.

## 4. User Stories

### 4.1 Management / Big Group Member (L1)

As a **Manager**, I want to:

* Receive **Summary Conclusions** to quickly understand the database health status.
* See **Impact Assessment** to decide whether to intervene and coordinate resources.
* **NOT** see complex SQL statements or details to avoid information overload.

### 4.2 Technical Lead (L2)

As a **Technical Lead**, I want to:

* See **SQL Hash** to quickly locate the problematic SQL.
* See **Changes in Key Metrics** to assess the impact of performance fluctuations on the business.

### 4.3 DBA / Private Group Member (L3)

As a **DBA**, I want to:

* View **SQL Fragments** or **Full Analysis Reports** for in-depth diagnosis and optimization.
* Receive **Detailed Alert Information** to respond to database anomalies immediately.
* Receive **Index Optimization Suggestions** to proactively improve system performance before issues arise.

### 4.4 SysOps

As a **System Operations Engineer**, I want:

* The Push Service to run as an **Independent Process**, without modifying the existing PoWA architecture.
* The service to have **Read-only Permissions**, ensuring database security.

## 5. Compliance & SecurityRequirements

* **Data Scope**: Only processes performance metadata.
* **System Positioning**: O&M monitoring auxiliary system.
* **Risk Assessment**: Low risk and highly controllable.

## 6. Roadmap

1. **Configuration Enhancement**: Support configurable rules (YAML / DSL).
2. **Architecture Extension**: Support Multi-instance / Multi-tenancy.
3. **Smart Noise Reduction**: Implement push de-duplication and suppression.
4. **Correlation Analysis**: Correlate with release/change events.
