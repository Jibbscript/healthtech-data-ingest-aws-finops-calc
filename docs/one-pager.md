# Throne Ingest PoC One-Pager

**What it proves:** a small Go/AWS architecture can ingest health-device captures, process them asynchronously, expose operational telemetry, and keep unit economics visible from day one.

**Core path:** device gRPC upload → S3 raw bucket → SQS ingest queue → processor → inference service → Postgres findings + derived S3 artifact → REST API.

**Reliability posture:** SLOs cover ingest availability, ingest latency, freshness, and zero acknowledged-data loss. Dashboards surface RED metrics, queue backpressure, DLQ depth, inference p99, and error-budget burn.

**Cost posture:** the live Grafana cost dashboard converts real bytes/day and captures/day into current-burn $/user/month. Main levers are S3 lifecycle tiering, compression, ARM/Graviton fleet, Fargate right-sizing, and Spot processor workers.

**What is intentionally fake in the PoC:** synthetic inference model, no real PHI, single-region dev environment, simplified Grafana pricing constants.
