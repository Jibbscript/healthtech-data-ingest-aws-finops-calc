# Throne Ingest PoC — Phased Implementation Plan

**Purpose.** A phase-by-phase breakdown of the work needed to deliver the system described in `arc42-throne-ingest-poc.md` plus its interleaved cost calculator (PoC #1). Each phase is sized to a working chunk with a verifiable exit criterion. Atomic work units (issue-sized) get decomposed in a separate step — this doc stops one level above that.

**Total target.** 2 days of focused work for #2 core; +1 day for #1 (cost calculator) folded in as later phases. Phases 1–6 are #2; phases 7–9 are #1; phase 10 is integration and demo polish.

**Cadence convention.** Each phase exits when:
1. The acceptance check passes (manually verifiable in <2 min).
2. The repo is in a committable, demo-able state.
3. The next phase doesn't require revisiting this one.

This is intentional — phases must not be entangled, because the atomic-decomposition step downstream will need clean phase boundaries to slot work units against.

---

## Phase 0 — Repository scaffold (~30 min)

**Goal.** Empty but credible repo skeleton. Nothing runs yet.

**Deliverables:**
- Repo created (`throne-backend-poc` or similar)
- Go module initialized (`go 1.22`)
- Standard layout: `cmd/`, `internal/`, `pkg/`, `api/proto/`, `infra/`, `docs/`, `web/`, `scripts/`
- `Makefile` with stubbed targets (`make dev`, `make test`, `make build`, `make lint`, `make seed`, `make load`)
- `README.md` with the arc42 dossier linked and the §8.4 onboarding contract questions answered as TBD
- `.gitignore`, `.editorconfig`, `.golangci.yml`, `buf.yaml`, `buf.gen.yaml`
- `docker-compose.yml` with placeholders for postgres, localstack, prometheus, grafana, tempo
- License + ARC42 dossier committed to `docs/`

**Acceptance.** `make lint` runs and passes against an empty codebase. `docker-compose up` brings up postgres + localstack and they expose healthy ports.

**Non-goals.** Any actual service code; CI; AWS provisioning.

---

## Phase 1 — Proto schemas and generated code (~1 hr)

**Goal.** Canonical wire format defined, generated code committed.

**Deliverables:**
- `api/proto/throne/v1/capture.proto` — `IngestService.Capture` (streaming) + `CaptureAck`
- `api/proto/throne/v1/inference.proto` — `InferenceService.Infer` + types
- `api/proto/throne/v1/api.proto` — REST gateway annotations for mobile-facing endpoints (`GET /v1/users/{id}/captures`, `GET /v1/captures/{id}`)
- `api/proto/throne/v1/types.proto` — shared types (Device, Capture, Finding, User)
- `buf` configured; `buf lint`, `buf breaking` against main, `buf generate` wired into Makefile
- Generated Go code committed under `internal/gen/`
- Generated OpenAPI/Swagger committed under `api/openapi/`

**Acceptance.** `make proto` regenerates cleanly; `buf lint` and `buf breaking` both pass; generated Go compiles standalone.

**Non-goals.** Actual service implementations.

---

## Phase 2 — Terraform foundations (~2 hr)

**Goal.** Infrastructure modules written and applied to a sandbox AWS account (or fully localstack-emulatable).

**Deliverables:**
- `infra/modules/network/` — VPC, public/private/data subnets, NAT, S3 + SQS + SSM endpoints, baseline SGs
- `infra/modules/storage/` — S3 bucket module taking `name`, `data_class`, `lifecycle_rules` as inputs; KMS key per data class
- `infra/modules/queue/` — SQS queue + DLQ + redrive policy; configurable visibility timeout, retention
- `infra/modules/data/` — RDS Postgres (db.t4g.small in dev); subnet group, parameter group, automated backups
- `infra/modules/service/` — generic ECS service module taking taskdef inputs, autoscaling config, target group, log group
- `infra/modules/observability/` — CloudWatch log groups, baseline alarms (DLQ depth, error rate), Grafana datasources as code
- `infra/envs/dev/main.tf` — composes the above for a single dev environment
- Backend state in S3 + DynamoDB lock table (or local backend for PoC if AWS unavailable)
- `tflint`, `tfsec` configs; both pass

**Acceptance.** `cd infra/envs/dev && terraform apply` succeeds against either real AWS or localstack. Resources visible. State file written.

**Non-goals.** Prod env; CI Terraform automation; cross-account roles.

---

## Phase 3 — throne-ingest service (~2 hr)

**Goal.** Working gRPC ingest server with REST gateway, accepting captures, writing to S3, enqueueing to SQS.

**Deliverables:**
- `cmd/throne-ingest/main.go` — wiring, config, graceful shutdown
- `internal/ingest/server.go` — gRPC server impl of `Capture`
- `internal/ingest/storage.go` — S3 client wrapper (uses `aws-sdk-go-v2`)
- `internal/ingest/queue.go` — SQS publisher wrapper
- `internal/ingest/auth.go` — mTLS cert validation, device_id extraction
- `internal/ingest/middleware/` — OTel tracing, structured logging, request metrics
- `docker/service.Dockerfile` — shared multi-stage, distroless final image builder for Go services
- REST gateway endpoint via `grpc-gateway` (separate listener on port 8080)
- Unit tests for handler logic, idempotency, error paths
- Integration test: docker-compose-up, send capture via grpcurl, assert S3 object + SQS message

**Acceptance.** `make dev` brings up ingest; `grpcurl` upload of a 5 MB blob succeeds in <500 ms locally; S3 has the object; SQS has the message; OTel trace appears in tempo.

**Non-goals.** Anything downstream of SQS.

---

## Phase 4 — throne-processor + throne-inference (~3 hr)

**Goal.** Async pipeline consuming from SQS, calling inference, writing findings.

**Deliverables:**
- `cmd/throne-processor/main.go` + `internal/processor/`
  - SQS consumer with proper visibility-timeout / extended-receive handling
  - S3 reader for raw payload
  - gRPC client to inference, with deadline + retry policy
  - Postgres writer (using `pgx`, not `database/sql`)
  - Idempotency check against captures table
  - Structured error handling: transient → no-ack, permanent → ack + mark failed
- `cmd/throne-inference/main.go` + `internal/inference/`
  - gRPC server impl of `Infer`
  - Stub model: deterministic synthetic findings from input hash (no real ML)
  - Configurable simulated latency (default 1s, configurable to 15s for load tests)
- Database migrations (`internal/db/migrations/`) — `users`, `devices`, `captures`, `findings`, `audit_log`
- `golang-migrate` wired into `make migrate`
- Trace context propagation through SQS message attributes (custom, not built-in)
- Integration test: full happy path from grpcurl ingest → finding row in postgres

**Acceptance.** `make seed` creates a device + user; `make load` (light) sends 100 captures; all 100 land as `succeeded` findings in postgres within 60s.

**Non-goals.** Real ML; the PHI redaction sidecar (slot exists, passthrough only); throne-api.

---

## Phase 5 — throne-api + load generator (~2 hr)

**Goal.** Mobile-facing API service and a load generator that proves scaling claims.

**Deliverables:**
- `cmd/throne-api/main.go` + `internal/api/`
  - REST endpoints: `GET /v1/users/{id}/captures`, `GET /v1/captures/{id}`, `GET /v1/findings/{id}`
  - JWT validation middleware (HS256 in PoC, RS256 production-shaped)
  - Signed-URL issuance for image retrieval (`POST /v1/captures/{id}/image-url`)
  - Same OTel/log/metrics conventions as other services
- `cmd/throne-load/main.go` — configurable load generator
  - Flags: `--devices N`, `--captures-per-device-per-day N`, `--duration`, `--target`
  - Simulates device fleet using goroutines + ticker
  - Emits its own metrics so its load can be charted alongside service metrics
- Light load test runs from CI nightly (configurable, defaults to off for PoC)

**Acceptance.** `make load -- --devices 100 --captures-per-device-per-day 10 --duration 1m` runs cleanly; metrics show ingest rate, queue depth, processing rate; all captures land as findings; no DLQ messages.

**Non-goals.** Stress-test at 2 TB/day (theoretical exercise here, not literal).

---

## Phase 6 — Observability stack & SLO doc (~2 hr)

**Goal.** Dashboards, alerts, SLO doc — the §10 of the arc42 made real.

**Deliverables:**
- `infra/modules/observability/dashboards/` — Grafana JSON for: Overview, Pipeline Health, Cost Dashboard (stub at this phase, real data in phase 9)
- Alert rules: ingest error rate > 1%, queue age > 60s, DLQ depth > 0, inference p99 > 30s, postgres connection saturation > 80%
- `docs/slo.md` — the SLO doc from §10.2 of the arc42, formatted as standalone doc
- `docs/runbooks/` — one runbook per alert; format: symptom → check → mitigation → root cause
- Chaos toy: `scripts/chaos.sh` that can `--kill processor` (stops a task), `--saturate queue` (sends 10k bursts), `--lag inference` (sets inference simulated latency to 30s); used to demonstrate graceful degradation

**Acceptance.** Open grafana, see real metrics from `make load` run. Run `scripts/chaos.sh --kill processor`. Observe queue age alert fire. Recover. Observe error budget burn logged.

**Non-goals.** PagerDuty integration; cost-anomaly alerts (deferred to phase 9).

---

*Phases 1–6 deliver build #2 in full. Below is the cost calculator (PoC #1), folded as continuation.*

---

## Phase 7 — Pricing API wrapper (~1.5 hr)

**Goal.** Backend service for the cost calculator: pulls AWS Pricing API data, caches it, exposes a narrow interface.

**Deliverables:**
- `cmd/throne-cost-api/main.go` + `internal/cost/`
  - HTTP server (REST/JSON, no gRPC — this is a tool, not a service)
  - AWS Pricing API client (calls `pricing.GetProducts` for the SKUs we care about)
  - Cache: 24h TTL, persisted to local file + S3 in prod
  - Endpoints:
    - `GET /v1/pricing/fargate?region=us-east-1` — vCPU-hour and GB-hour pricing
    - `GET /v1/pricing/s3?region=us-east-1&class=standard|ia|glacier` — storage pricing
    - `GET /v1/pricing/rds?instance=db.m6g.large&region=us-east-1` — instance pricing
    - `GET /v1/pricing/sqs?region=us-east-1` — request pricing
    - `GET /v1/pricing/data-transfer?from=us-east-1&direction=out-internet` — egress
  - Fallback to bundled snapshot when Pricing API unreachable
  - The pricing cache itself is the bundled snapshot — generated by a `make refresh-pricing` target

**Acceptance.** `curl localhost:9000/v1/pricing/fargate?region=us-east-1` returns valid JSON with vCPU and GB hourly rates that match AWS published prices (verifiable manually).

**Non-goals.** Reserved instance / Savings Plan modeling (out of scope for PoC).

---

## Phase 8 — Cost model & calculator UI (~3 hr)

**Goal.** React frontend for the live cost calculator, wired to the pricing API.

**Deliverables:**
- `web/cost/` — Vite + React + TypeScript + Tailwind + shadcn/ui (minimal styling — locked to defaults per the upstream constraint of "don't bikeshed UI")
- Components:
  - `<InputPanel/>` — all 12 input parameters from §10.3, with sensible defaults
  - `<CostBreakdown/>` — pie/stacked-bar of monthly cost by category
  - `<PerUserDisplay/>` — large numeric, color-coded against $6 line
  - `<ProjectionChart/>` — 24-month stacked area, accounting for storage accumulation
  - `<ScenarioToggles/>` — with/without tiering, with/without Spot, with/without compression — shows deltas
  - `<ExportButton/>` — generates a one-page PDF summary (using `html2canvas` + `jspdf`)
- Cost model itself lives in `web/cost/src/model/` — pure TypeScript, unit-tested with vitest
- All math in one file: `cost-model.ts`; tests in `cost-model.test.ts` cover the boundary cases (300 DAU, 100k DAU, with/without each lever)

**Acceptance.** Open http://localhost:5173, change DAU to 100000, watch the $/user/month number jump. Toggle off tiering — watch it go red. Toggle back on — green. Export PDF, open it, looks presentable.

**Non-goals.** Auth on the calculator (it's a tool, not a product); multi-user state; saving scenarios server-side.

---

## Phase 9 — Live cost dashboard wired to real telemetry (~1.5 hr)

**Goal.** Close the loop: the cost dashboard panel in Grafana shows real, current-burn $/user/month from real ingest metrics.

**Deliverables:**
- Update `infra/modules/observability/dashboards/cost.json` — Grafana dashboard panels:
  - Panel 1: current captures/day (from metric)
  - Panel 2: current bytes ingested/day (from metric)
  - Panel 3: extrapolated $/month at current burn rate (computed in Grafana via transforms + pricing constants)
  - Panel 4: $/user/month — needs current DAU input (set as Grafana template variable)
  - Panel 5: cost projection 24 months out — uses same model as the React calculator (the math lives in two places, intentionally — a small price for not having Grafana call the cost-api at every render)
- Document in `docs/cost-runbook.md`: how to read the dashboard, when to escalate, what each lever does

**Acceptance.** Run a sustained load test for 5 min. Cost dashboard reflects the burn rate. Numbers roughly match what the standalone calculator would produce given the same inputs.

**Non-goals.** Anomaly detection on cost (would be nice; out of scope).

---

## Phase 10 — Integration, demo polish, README pass (~2 hr)

**Goal.** Repo is ready to hand to a hiring committee.

**Deliverables:**
- Top-level `README.md` rewritten with:
  - 30-second elevator: what this is and why it exists
  - 60-second demo path: `make demo` brings everything up and runs a scripted scenario
  - Architecture diagram (link to §5.1 of arc42)
  - "How to evaluate me from this repo": pointer list to the load-bearing files
  - Honest "what's not real" section — model is a stub, no real PHI, etc.
- `make demo` target — orchestrates: `make dev && make seed && make load -- --duration 30s && open http://localhost:3000 && open http://localhost:5173`
- A short walkthrough video script (`docs/demo-script.md`) — not the video itself, but the script for what to say during a 5-min screen-share
- One-pager PDF (`docs/one-pager.pdf`) — exec summary suitable for emailing to the recruiter before the interview
- All ADRs from arc42 §9 broken out into `docs/adr/000X-*.md` files
- All TODOs in code resolved or explicitly tagged with `// TODO(post-poc):`

**Acceptance.** A friend who has never seen the repo can clone, run `make demo`, and reach a working state in <5 min, and after reading the README has a defensible answer to "what does this person know about backend architecture."

**Non-goals.** Production-readiness in any literal sense.

---

## Phase summary table

| Phase | Title | Est. time | Cumulative | Maps to arc42 |
|---|---|---|---|---|
| 0 | Repo scaffold | 0.5h | 0.5h | — |
| 1 | Proto schemas | 1h | 1.5h | §8.3 |
| 2 | Terraform foundations | 2h | 3.5h | §7.3, §9 (multiple ADRs) |
| 3 | throne-ingest | 2h | 5.5h | §5.2 |
| 4 | processor + inference | 3h | 8.5h | §5.3, §5.4 |
| 5 | throne-api + load gen | 2h | 10.5h | §5.5, §9 ADR-008 |
| 6 | Observability + SLO | 2h | 12.5h | §10.2 |
| 7 | Pricing API wrapper | 1.5h | 14h | §10.3 |
| 8 | Calculator UI | 3h | 17h | §10.3 |
| 9 | Live cost dashboard | 1.5h | 18.5h | §10.3 |
| 10 | Demo polish | 2h | 20.5h | §8.4 |

**Total: ~20 hours** — fits in a focused weekend or 3 spread weekday evenings + a Saturday.

---

## Risks specific to this plan

| Risk | Likelihood | Mitigation |
|---|---|---|
| Phase 2 (Terraform) explodes — AWS sandbox issues, IAM hell | M-H | Have localstack fallback ready; commit to "real AWS only if it doesn't cost a day" |
| Phase 4 (processor + inference) ambitious for 3 hours | M | If pinched, cut the inference service into the processor and refactor in a follow-up; ADR-trail the decision |
| Phase 8 (UI) becomes a styling rabbit-hole | H | Hard rule: shadcn defaults, no custom CSS, no animation polish. If a component looks ugly, that's fine. |
| Phase 9 cost-dashboard math disagrees with calculator | M | Pull math out into a shared constants file referenced by both; accept that two consumers exist |
| Adversarial reviewers spot the stub model | L | Lead with it — the README explicitly names the model as a stub. The architecture, not the model, is the artifact. |

---

## What the next decomposition step needs to produce

The atomic work-unit decomposition (separate task) should produce, for each phase above:

- A list of GitHub-issue-sized tasks (each ≤ 2 hours, single-PR-shaped)
- Explicit dependencies between tasks within and across phases
- A "definition of done" per task tighter than the phase acceptance criterion
- Optional: estimate hours per task and identify the critical path

This doc deliberately stops one level above that — phases are interview-narrative-sized; atoms are commit-sized.
