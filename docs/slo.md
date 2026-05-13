# Service-Level Objectives

These SLOs cover the PoC ingest pipeline and are intentionally small enough for a two-person on-call rotation to operate.

| SLO | Target | Window | Measurement | Error budget |
|---|---:|---|---|---:|
| Ingest availability | 99.9% successful uploads | 30d rolling | `2xx / all` for `throne_ingest_requests_total` | 43 min/month |
| Ingest latency | p99 < 500 ms | 30d rolling | `throne_ingest_request_duration_seconds` | latency objective |
| Freshness | p95 capture-to-finding < 60s | 30d rolling | queue age + processor completion metrics | freshness objective |
| Pipeline durability | zero acknowledged-data loss | always | DLQ + audit reconciliation | 0 |

## Error-budget policy

- Fast burn: page when 2% of monthly availability budget is consumed in 1 hour.
- Slow burn: create an incident ticket when 10% of monthly budget is consumed in 6 hours.
- During budget exhaustion, feature work that increases request volume or pipeline complexity pauses until the owning team closes the incident review.

## Required signals

- RED metrics per HTTP/gRPC service: rate, errors, duration.
- Queue signals: visible depth, oldest age, DLQ depth, retry/redrive count.
- Inference signals: p50/p95/p99 duration, inflight requests, error rate by model version.
- Storage cost proxy: bytes written by bucket and data class.
- Postgres saturation: connection-pool use and RDS `DatabaseConnections`.

Dashboards: `Throne Overview`, `Throne Pipeline Health`, and `Throne Live Cost Dashboard`.
