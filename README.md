# Throne Ingest PoC + FinOps Calculator

A production-shaped backend interview artifact for a healthtech ingest pipeline: device captures arrive through an ingest service, are queued, processed through a deterministic inference stub, persisted, exposed through a read API, and tied to a live cost calculator.

## 30-second elevator

This repository demonstrates the parts of backend architecture that matter for a high-volume healthtech data path: async durability, idempotency, mTLS/JWT-shaped trust boundaries, S3/SQS/Postgres data separation, observability, Terraform-shaped infrastructure, and unit economics against a $6/user/month constraint. The model is intentionally synthetic; the architecture and cost controls are the artifact.

## 60-second demo path

```bash
make build
make demo
make run-cost-api
cd web/cost && npm install && npm run dev
```

Then open:

- Cost API: <http://localhost:9000/v1/pricing/fargate?region=us-east-1>
- Calculator UI: <http://localhost:5173>
- Grafana (optional): `docker compose up -d grafana prometheus tempo` then <http://localhost:3001>

## Load-bearing files for evaluators

- `api/proto/throne/v1/*.proto` — service contracts.
- `internal/ingest`, `internal/processor`, `internal/inference`, `internal/api` — service implementations and tests.
- `infra/modules/**` and `infra/envs/dev` — AWS/Terraform shape.
- `web/cost/src/model/cost-model.ts` — standalone FinOps math.
- `docs/slo.md`, `docs/runbooks/**`, `docs/cost-runbook.md`, `docs/adr/**` — ops and architecture decision trail.

## What is intentionally not real

- No real PHI or biological image classifier is included.
- Local mode uses file-backed S3/SQS/Postgres-shaped adapters so reviewers can run the pipeline without cloud credentials.
- The protobuf surface is canonical: `make proto` runs `buf generate` for Go, gRPC, grpc-gateway, and OpenAPI artifacts. Local demo HTTP endpoints remain for clone-and-run compatibility.
- Pricing API endpoints use a bundled May 2026 snapshot when live AWS Pricing credentials are unavailable.

## Commands

```bash
make help          # list targets
make proto         # regenerate committed Go/gRPC/gateway/OpenAPI artifacts
make test          # Go unit/integration-shaped tests
make build         # all Go commands
make lint          # gofmt + tests + terraform fmt when available
make docker-config # validate docker-compose syntax
docker build -f docker/service.Dockerfile --build-arg SERVICE=throne-ingest .
```

## Onboarding contract answers (arc42 §8.4)

- **How do I run it?** `make demo` for local data generation; individual `make run-*` targets for services.
- **Where is state?** `.data/` for local file-backed object storage, queue messages, and JSON DB.
- **How is production different?** Terraform modules replace local adapters with S3, SQS, RDS, ECS/Fargate, CloudWatch/Grafana.
- **What should a reviewer focus on?** Service boundaries, failure/idempotency paths, observability/runbooks, and cost-model levers.

## Architecture diagram

```text
Device -> throne-ingest -> S3(raw) + SQS -> throne-processor -> throne-inference
                                                  |                 |
                                                  v                 v
                                             Postgres           S3(derived)
Mobile/Web -> throne-api -> Postgres + signed S3 URL
FinOps UI -> throne-cost-api -> AWS Pricing snapshot/live cache
```
