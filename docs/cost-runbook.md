# Cost Dashboard Runbook

Use `Throne Live Cost Dashboard` to connect real ingest telemetry to unit economics.

## How to read it

1. Set the `DAU` template variable to the active-user assumption for the scenario.
2. Compare current captures/day and bytes/day against the load-test or production expectation.
3. Read extrapolated storage dollars/month and $/user/month as current-burn estimates, not invoices.
4. Cross-check with the standalone calculator for scenario planning; Grafana uses simplified constants to stay render-only.

## Escalation thresholds

- Green: ≤ $6/user/month. No action.
- Amber: > $6 and ≤ $8/user/month. File a cost review issue within one business day.
- Red: > $8/user/month or sudden 2x bytes/day. Page the owner during business hours and pause synthetic load.

## Primary levers

- S3 lifecycle tiering: largest storage lever; keep Standard → IA at 30d and IA → Glacier IR at 90d unless product needs hot history.
- Compression: reduces S3, data transfer, and processing cost together.
- ARM/Graviton fleet: ~20% off Fargate compute across ingest, processor, inference, and API on a single Go recompile.
- Fargate right-sizing: profile p95 CPU/memory and set requests with ~20% headroom.
- Fargate Spot: safe for processor workers because SQS redelivers; do not use for latency-sensitive inference without canary data.

## Diagnose a cost jump

Check bytes/day first. If flat, check DAU variable and pricing constants. If bytes rose, split by bucket/class and compare recent deploys or load tests. If compute rose, inspect task counts and autoscaling events.
