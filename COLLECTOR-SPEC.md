# MindSuite Discovery Collector — Specification

The Discovery Collector is the part of MindSuite that scans what the browser cannot reach: **databases, the environment and infrastructure, messaging and caching, identity, and telemetry**. It runs **inside the client's network**, uses **read-only** access, collects **metadata only**, and writes one JSON **bundle**. The client reviews the bundle, then imports it on **01 Record evidence → Connect sources → Import collector bundle**.

**Status:** not built yet. The web app already imports this format (`importBundle()` in `index.html`). Use **Record evidence → How collection works → Download sample bundle** to get a complete example.

---

## 1. Principles (non-negotiable)

1. **Read-only.** Never change anything in a target system. Use dedicated read-only accounts (section 4).
2. **Metadata only.** No business data rows, no file contents, no personal data. Counts and sizes are allowed.
3. **No secrets.** Never write passwords, tokens, keys or connection-string credentials to the bundle or to logs. Strip `user:pass@` from URLs.
4. **Runs locally.** It needs no internet access. The bundle is written to disk; the client decides whether to hand it over.
5. **Transparent.** It writes a human-readable run log (targets, timings, permission errors), and has a `--dry-run` that lists what it would query.
6. **Licence-aware.** Only use licensed diagnostic features (e.g. Oracle AWR/ASH) when the client passes `--oracle-diagnostics-licensed`.

## 2. Shape of the tool

- A single command-line tool (suggested: Python 3.11, packaged as one executable and as a container image).
- Configuration: a YAML file listing targets and which modules to run. Credentials come from environment variables or the OS keychain, never from the YAML.
- Command: `mindsuite-collector run --config targets.yaml --out bundle.json`.
- Modules: `database`, `infrastructure`, `middleware`, `identity`, `telemetry`. Each is optional; a bundle may contain any subset.

## 3. Bundle format (version 1)

```json
{
  "mindsuiteBundle": 1,
  "collectorVersion": "0.1.0",
  "collectedAt": "2026-11-20T10:00:00Z",
  "collectedBy": "ops team / person who ran it",
  "environment": "production",

  "database": {
    "engine": "oracle | sqlserver | postgresql | mysql | mariadb | db2 | mongodb",
    "version": "19c", "host": "db-prod-01",
    "tables":   [{ "schema": "HL", "name": "GROWER", "rows": 182000, "sizeMb": 1900 }],
    "routines": [{ "name": "PKG_SETTLEMENT", "type": "PACKAGE | PROCEDURE | FUNCTION | TRIGGER | VIEW", "lines": 4200 }],
    "jobs":     [{ "name": "NIGHTLY_SETTLEMENT_PREP", "schedule": "Daily 01:30" }],
    "dbLinks":  [{ "name": "FINANCE_GL", "target": "finance-db-01" }],
    "users":    [{ "name": "HL_APP", "privileges": ["DBA"] }]
  },

  "infrastructure": {
    "hosts": [{ "name": "app-prod-01", "os": "Red Hat Enterprise Linux 7.9", "cpu": 4, "ramGb": 16,
                "virtualization": "VMware vSphere 6.7 | Hyper-V | KVM | (empty = physical)",
                "role": "application", "software": ["Apache Tomcat 8.5", "Java 11"] }],
    "connections": [{ "from": "app-prod-01", "to": "db-prod-01", "port": 1521 }],
    "kubernetes":  [{ "distribution": "OpenShift | AKS | EKS | GKE | Kubernetes", "version": "1.27", "namespaces": 12, "workloads": 48 }],
    "cloud":       [{ "provider": "aws | azure | gcp", "account": "id or name", "regions": ["eu-west-1"], "resources": 214 }]
  },

  "middleware": {
    "brokers": [{ "type": "IBM MQ | Kafka | RabbitMQ | ActiveMQ | Azure Service Bus | Amazon SQS", "name": "QM_HL",
                  "version": "9.1", "queues": ["HL.LOT.GRADED"], "messagesPerSecond": 6 }],
    "caches":  [{ "type": "Redis | Memcached", "name": "cache-01", "version": "5.0",
                  "evictionPolicy": "allkeys-lru", "hitRatio": 0.72, "maxMemoryMb": 4096 }]
  },

  "identity": {
    "provider": "Active Directory | Microsoft Entra ID | Okta | Ping | Keycloak | LDAP",
    "protocols": ["LDAP", "SAML", "OpenID Connect", "Forms login"],
    "mfaEnforced": false, "groups": 34, "serviceAccounts": 6, "sharedAccounts": 3,
    "staleAccounts": 41, "privilegedAccounts": 9,
    "findings": [{ "title": "Application stores its own password hashes (SHA-1)", "severity": "high | medium | low",
                   "recommendation": "Move authentication to the identity provider" }]
  },

  "telemetry": {
    "window": "2022-01-01 to 2022-12-31",
    "avgTps": 3.2, "peakTps": 48, "peakTimes": "Weekdays 07:00–11:00", "peakMonths": [7, 8, 9, 10],
    "p95Ms": 2400, "errorRate": 0.014, "users": 410, "concurrentUsers": 130,
    "unusedFeatures": ["Legacy grower export"]
  }
}
```

**What the app does with each section** (see `importBundle()`):
- **database:** tables become data stores (largest 60); stored code and jobs become components; database links become interfaces; summed size becomes measured database size; admin accounts and large stored-code volume become risks.
- **infrastructure:** sets the platform and hosting findings; virtualization, Kubernetes, cloud, OS and server software become technologies; connections to hosts outside the bundle become interfaces; out-of-support operating systems become risks.
- **middleware:** brokers become interfaces and technologies; caches become data stores; a low hit ratio or `noeviction` becomes a risk.
- **identity:** the provider and protocols become a technology; no MFA, shared, inactive and privileged accounts and listed findings become security risks.
- **telemetry:** becomes the system-wide **measured capacity profile** (used by every context until it gets its own); a high error rate and unused features (retire candidates) become risks.

Everything imported is marked **origin `collector`, status "Extracted"**, and waits for a person to confirm it. A new import replaces earlier unconfirmed collector records and never overwrites anything a person confirmed.

## 4. What each module reads (read-only)

### database
| Engine | Catalog sources | Minimum permission |
|---|---|---|
| Oracle | `ALL_TABLES`, `ALL_TAB_COLUMNS`, `ALL_CONSTRAINTS`, `ALL_SOURCE`, `ALL_VIEWS`, `ALL_TRIGGERS`, `ALL_DEPENDENCIES`, `DBA_SEGMENTS` (sizes), `DBA_SCHEDULER_JOBS`, `DBA_DB_LINKS`, `DBA_ROLE_PRIVS`/`DBA_SYS_PRIVS` | `CREATE SESSION` + `SELECT_CATALOG_ROLE` (or `SELECT ANY DICTIONARY`) |
| SQL Server | `sys.tables`, `sys.columns`, `sys.foreign_keys`, `sys.sql_modules`, `sys.sql_expression_dependencies`, `sys.dm_db_partition_stats`, `msdb.dbo.sysjobs`, linked servers, `sys.server_role_members` | `VIEW DEFINITION`, `VIEW SERVER STATE`, read on `msdb` job tables |
| PostgreSQL | `information_schema`, `pg_class`, `pg_proc`, `pg_depend`, `pg_total_relation_size`, `pg_roles`; `pg_stat_statements` if installed | `pg_read_all_stats` (+ `pg_monitor`) |
| MySQL / MariaDB | `information_schema` (TABLES, ROUTINES, TRIGGERS, EVENTS), `mysql.user` privileges | `SELECT` on `information_schema`, `PROCESS` |
| Db2 | `SYSCAT.TABLES`, `SYSCAT.ROUTINES`, `SYSCAT.ROUTINEDEP`, `SYSCAT.DBAUTH` | `SELECT` on `SYSCAT` |

Never `SELECT` from business tables. Row counts come from catalog statistics.

### infrastructure
- **VMware:** vSphere API with a read-only role (VMs, CPU, RAM, OS, cluster, datastore). An RVTools export is an accepted alternative input.
- **Kubernetes / OpenShift:** a read-only ClusterRole (`get`, `list` on namespaces, deployments, statefulsets, services, ingresses, routes, configmap **names only**, HPAs).
- **Cloud:** AWS read-only/SecurityAudit role (AWS Config), Azure Reader (Resource Graph), GCP Viewer (Cloud Asset Inventory).
- **Servers:** SSH (Linux/Unix) or WinRM (Windows) with a non-admin account running read-only commands: OS and version, CPU/RAM, virtualization (`systemd-detect-virt`, `dmidecode -s system-product-name` where permitted, `Win32_ComputerSystem`), installed and running software (process list), listening ports, and **established connections** (`ss -tnp` / `netstat -ano` / `Get-NetTCPConnection`), aggregated to host:port pairs.

### middleware
- **IBM MQ:** `runmqsc` display commands (queue managers, queues, depths, channels) with a display-only authority.
- **Kafka:** Admin API (topics, partitions, consumer groups) with describe-only ACLs.
- **RabbitMQ:** Management HTTP API with a `monitoring` tag user.
- **Azure Service Bus / Amazon SQS:** through the cloud read-only roles above.
- **Redis:** `INFO` (memory, keyspace hits and misses → hit ratio) and `CONFIG GET maxmemory-policy` if allowed. Never read keys.

### identity
- **Active Directory / LDAP:** read-only bind: group count, service-account and privileged-group membership counts, accounts inactive for 90+ days. No password attributes.
- **Microsoft Entra ID / Okta:** read-only API (directory read, policy read): app registrations, MFA / conditional-access policies, sign-in method usage.
- **Database grants:** from the database module.

### telemetry
- **APIs:** Datadog, Dynatrace, New Relic, Azure Monitor, CloudWatch or Prometheus, read-only API key, aggregated to requests per hour for the window, p95 latency, error rate and peaks.
- **Alternative:** a CSV export with `timestamp, requests, errors, p95_ms`; the collector aggregates it.
- **Feature usage:** page/route request counts; routes with zero use in the window become `unusedFeatures`.

## 5. Mainframe and IBM i

These need the client's own platform teams: source-library exports (Endevor, ChangeMan, PDS members), job scheduler exports, RACF/ACF2/Top Secret reports, and Db2 for z/OS catalog queries. Plan them as **exports** in the bundle, not live scans.

## 6. Acceptance tests for the collector

1. Importing its output into the app produces no console errors, and the "What the agent found" table on Record evidence shows the expected views as detected.
2. A grep of the bundle for the passwords used in the test environment returns nothing.
3. Running it with an account that lacks a permission produces a clear log line and a partial bundle, not a crash.
4. Two runs on an unchanged environment produce the same bundle (apart from `collectedAt`).
