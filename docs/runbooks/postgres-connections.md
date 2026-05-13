# Runbook: Postgres connection saturation > 80%

**Symptom:** `PostgresConnectionSaturation` fires; API and processor writes may fail.

**Check:** Inspect RDS `DatabaseConnections`, app pool sizes, slow queries, and current task counts.

**Mitigation:** Reduce per-task pool size, restart leaking tasks, and temporarily scale write-heavy services down if the database is rejecting connections.

**Root cause path:** Identify the service consuming connections, inspect query patterns, and decide whether to tune pools, add pgbouncer, or scale RDS.
