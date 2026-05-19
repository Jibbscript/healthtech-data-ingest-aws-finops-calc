# Throne Ingest PoC — Atomic Work Decomposition

**Companion to** `phased-plan.md`. Each atom is single-PR-shaped, ≤90min target, ≤120min ceiling. Format per atom:

```
### N.M — title [estimate-minutes] [deps]
- brief scope
- DoD: definition of done (tighter than phase acceptance)
```

deps notation: `→A.B` means "depends on A.B". no entry = no in-plan deps beyond same-phase predecessors.

estimates are 50%-confidence — multiply by ~1.4 for 80% confidence per standard estimation correction.

---

## phase 0 — repo scaffold

### 0.1 — init repo + go module [10m]
- `git init`, `go mod init github.com/jibbscript/throne-backend-poc`, license file
- DoD: `go build ./...` succeeds against empty package; pushed to remote

### 0.2 — standard directory layout [10m] [→0.1]
- create `cmd/`, `internal/`, `pkg/`, `api/proto/`, `infra/{modules,envs/{local,dev}}`, `docs/`, `web/`, `scripts/`, `test/`
- empty `.gitkeep` in each, README stubs in `infra/` and `docs/`
- DoD: `tree -L 2` matches the arc42 §7.3 layout exactly

### 0.3 — root Makefile with stubbed targets [20m] [→0.2]
- targets: `dev`, `test`, `lint`, `proto`, `build`, `seed`, `load`, `migrate`, `demo`, `refresh-pricing`
- each prints "TODO: <phase X>" with exit 0
- DoD: `make help` lists all targets with one-line descriptions

### 0.4 — docker-compose skeleton [25m] [→0.2]
- services: postgres:16, localstack (S3+SQS+SSM), prom/prometheus, grafana/grafana, grafana/tempo
- volumes for postgres data and grafana provisioning
- healthchecks on postgres and localstack
- DoD: `docker compose up -d && docker compose ps` shows all services healthy in <60s; `psql -h localhost` connects; `aws --endpoint http://localhost:4566 s3 ls` returns

### 0.5 — linter configs [20m] [→0.2]
- `.golangci.yml` (revive, errcheck, ineffassign, govet, staticcheck, gocyclo)
- `.editorconfig`, `.gitignore` (go + tf + node)
- `buf.yaml`, `buf.gen.yaml` stubs
- DoD: `make lint` exits 0 against empty codebase; CI matrix file written but not yet active

### 0.6 — docs/ copy + index [10m] [→0.2]
- copy arc42 dossier + phased plan + atomic decomp into `docs/`
- `docs/README.md` index linking each
- DoD: each linked doc renders correctly when viewed via github

---

## phase 1 — proto schemas

### 1.1 — buf config + tooling [15m] [→0.5]
- `buf.yaml` with `breaking.use: [FILE]`, `lint.use: [STANDARD]`
- `buf.gen.yaml` with go, go-grpc, grpc-gateway, openapiv2 plugins
- `scripts/install-proto-tools.sh` (idempotent install of buf + protoc plugins)
- DoD: `buf --version` works; `make proto` runs without error against empty proto dir

### 1.2 — types.proto [20m] [→1.1]
- `api/proto/throne/v1/types.proto`: Device, User, Capture, Finding, FindingResult, AuditEvent
- timestamps use `google.protobuf.Timestamp`; IDs are strings (UUIDs)
- DoD: `buf lint` passes; field tags reserved appropriately for future evolution

### 1.3 — capture.proto with streaming RPC [25m] [→1.2]
- `IngestService.Capture(stream CaptureChunk) returns (CaptureAck)`
- CaptureChunk = oneof{metadata, data_chunk}; first message must be metadata
- max chunk size 1 MB enforced by comment + runtime check later
- DoD: `buf lint` + `buf breaking` against empty baseline pass

### 1.4 — inference.proto [15m] [→1.2]
- `InferenceService.Infer(InferRequest) returns (InferResponse)`
- request carries s3_key, model_version, content_hash
- response carries Finding + confidence
- DoD: `buf lint` passes

### 1.5 — api.proto + grpc-gateway annotations [25m] [→1.2]
- `ThroneAPI` with `GetCapture`, `ListCaptures`, `GetFinding`, `CreateImageURL`
- google.api.http annotations for REST routes per arc42 §5.5
- DoD: `make proto` generates OpenAPI/Swagger v2 file at `api/openapi/throne.v1.swagger.json`

### 1.6 — generate + commit generated code [15m] [→1.3,1.4,1.5]
- run `make proto`; commit `internal/gen/` and `api/openapi/`
- DoD: generated code compiles; reflection enabled in generated server stubs

### 1.7 — CI gate for proto [20m] [→1.6]
- GitHub Actions workflow: lint, breaking-vs-main, ensure generated code matches
- DoD: PR that modifies a `.proto` without re-generating fails CI; PR that breaks an existing field fails

---

## phase 2 — terraform foundations

### 2.1 — TF backend + provider config [25m] [→0.5]
- `infra/envs/dev/{backend.tf,providers.tf,variables.tf,outputs.tf}`
- backend: S3 bucket + DynamoDB lock table (bootstrapped via one-time local script `scripts/bootstrap-tf-state.sh`)
- provider versions pinned
- DoD: `terraform init` succeeds; lock file committed

### 2.2 — network module [50m] [→2.1]
- VPC /16, 3 AZs, public/private/data subnets (/20 each)
- NAT gateway (single, dev), IGW
- S3 gateway endpoint (free), SSM + SQS interface endpoints
- baseline security groups: `sg-ingress-tasks`, `sg-data`, `sg-alb`
- DoD: `terraform plan` clean; resources tagged with `Project=throne`, `Env=dev`

### 2.3 — storage module [35m] [→2.1]
- input: bucket name, data class enum (phi|standard|telemetry), lifecycle rules list
- creates: KMS key (alias `alias/throne-<class>-<env>`), bucket, lifecycle config, public-access-block, bucket policy denying non-TLS
- emits outputs: bucket arn, kms key arn
- DoD: applied; `aws s3api get-bucket-encryption` returns the right KMS key

### 2.4 — queue module [25m] [→2.1]
- creates SQS queue + DLQ + redrive policy (maxReceiveCount=3)
- inputs: name, visibility timeout, retention
- DoD: applied; can send/receive a test message via CLI against localstack and against dev account

### 2.5 — data module (RDS) [40m] [→2.2]
- db.t4g.small in dev (small footprint, $25/mo)
- subnet group across data subnets, parameter group with `log_statement=ddl`
- automated backups 7d, deletion protection off in dev
- emits secret to SSM `/throne/dev/db/password` (random_password)
- DoD: `psql` connection from a bastion or session manager works; backup verified once

### 2.6 — service module (ECS) [50m] [→2.2]
- generic ECS Fargate service: taskdef, service, target group (optional), autoscaling (CPU + custom metric)
- inputs: name, image, cpu, memory, port, env, secrets refs, autoscaling min/max
- log group with 30d retention, OTel sidecar optional
- DoD: empty taskdef applies cleanly; outputs include service ARN + task role ARN

### 2.7 — observability module [40m] [→2.6]
- CloudWatch log groups for each service (created by service module, named here)
- alarms: DLQ depth > 0, ALB 5xx > threshold, RDS connections > 80%
- Grafana datasources provisioned via local file (`grafana/provisioning/datasources/`)
- DoD: alarms in OK state immediately after apply; grafana shows datasources

### 2.8 — dev env composition [30m] [→2.2,2.3,2.4,2.5,2.6,2.7]
- `infra/envs/dev/main.tf` instantiates each module with dev sizing
- variables file with environment defaults
- DoD: `terraform apply` from clean state succeeds in <10 min; `terraform destroy` succeeds

### 2.9 — tflint + tfsec CI [25m] [→2.8]
- GitHub Actions: lint, security scan, `terraform fmt -check`, `terraform validate`
- DoD: PR that breaks fmt fails; PR with insecure rule fails (e.g., public S3 bucket)

---

## phase 3 — throne-ingest service

### 3.1 — service skeleton + config loading [25m] [→1.6,0.4]
- `cmd/throne-ingest/main.go`: wire up logger, config, signal handling, graceful shutdown
- config via env vars with viper or stdlib; secrets from SSM in prod, dotenv in dev
- DoD: `make run-ingest` starts the server, logs "listening on :8443"; SIGTERM shuts down cleanly within 5s

### 3.2 — gRPC server with streaming Capture handler [45m] [→3.1]
- implements `IngestService.Capture` — accept stream, write to in-memory buffer up to limit, then to disk
- max payload size enforced (default 20 MB), chunk size validated
- DoD: unit test sends a 5 MB blob via in-process gRPC; handler returns CaptureAck with capture_id

### 3.3 — S3 client wrapper [25m] [→3.2,2.3]
- `internal/ingest/storage/s3.go` — wraps aws-sdk-go-v2/service/s3
- key format: `captures/<device_id>/<yyyy>/<mm>/<dd>/<capture_id>`
- DoD: integration test against localstack uploads a 5 MB blob and reads it back; checksum matches

### 3.4 — SQS publisher [20m] [→3.2,2.4]
- `internal/ingest/queue/sqs.go` — wraps SendMessage with retry + idempotency token
- message body is JSON: {capture_id, device_id, s3_key, captured_at}
- trace context injected into MessageAttributes
- DoD: integration test publishes; localstack ReceiveMessage returns it

### 3.5 — mTLS auth + device cert validation [40m] [→3.2]
- TLS listener with client-cert required
- thumbprint extracted from peer cert, looked up against allowlist (postgres or in-memory map for PoC)
- DoD: connection without cert refused; connection with allowlisted cert succeeds; cert thumbprint exposed to handler via context

### 3.6 — OTel instrumentation + slog handler [30m] [→3.2]
- otel SDK init, OTLP HTTP exporter to tempo (localhost:4318 in dev)
- slog handler emits JSON, attaches trace_id, drops fields matching PII denylist
- DoD: a Capture call produces a trace visible in tempo; logs in `docker logs throne-ingest` are JSON with trace_id

### 3.7 — REST gateway listener [25m] [→3.2,1.5]
- separate listener on :8080 running grpc-gateway mux
- maps `POST /v1/captures` to internal gRPC Capture (one-shot upload, not streaming)
- DoD: `curl -X POST localhost:8080/v1/captures` with multipart body returns 200 + capture_id

### 3.8 — Dockerfile + image push [25m] [→3.7]
- multi-stage build: golang:1.22 → distroless/static
- non-root user, healthcheck binary baked in
- pushed to ECR or local registry
- DoD: image is <30 MB; `docker run` boots in <2s

### 3.9 — integration test [35m] [→3.8]
- spins up docker-compose stack, uses grpcurl to send 100 captures
- asserts: 100 S3 objects, 100 SQS messages, all under 500ms p99
- DoD: test passes locally; runs as part of `make test-integration`

---

## phase 4 — processor + inference

### 4.1 — postgres schema migrations [30m] [→2.5,0.4]
- `internal/db/migrations/000001_initial.up.sql` — users, devices, captures, findings, audit_log
- unique constraints on (device_id, capture_uuid)
- indexes on (user_id, captured_at desc), (capture_id) on findings
- `golang-migrate` integration in `make migrate`
- DoD: `make migrate` applies clean; `\d+` in psql shows all tables; rollback works

### 4.2 — processor skeleton + SQS consumer loop [40m] [→3.4,4.1]
- `cmd/throne-processor/main.go` with long-poll receive (20s wait time), batch size 10
- worker pool: 5 goroutines, configurable
- graceful shutdown drains in-flight messages
- DoD: starts, polls, logs each receive; SIGTERM completes in-flight before exit

### 4.3 — S3 reader + idempotency check [30m] [→4.2,3.3]
- reads payload referenced by SQS message
- checks captures table for prior success on this capture_id; if found, ack and skip
- DoD: re-enqueuing same message acks without re-processing; new message processes through

### 4.4 — inference service skeleton [40m] [→1.6,2.6]
- `cmd/throne-inference/main.go` — gRPC server implementing `Infer`
- stub: deterministic finding from sha256(payload) seeding a rng
- configurable simulated latency via env var INFERENCE_LATENCY_MS (default 1000)
- exposes `/healthz` HTTP endpoint for ALB
- DoD: grpcurl call returns a Finding within configured latency ± 50ms

### 4.5 — processor → inference gRPC client [30m] [→4.3,4.4]
- gRPC client with connection pooling
- per-call deadline 30s, retry once on UNAVAILABLE (not on DEADLINE_EXCEEDED)
- trace context propagation
- DoD: end-to-end happy path test: SQS msg → processor → inference → finding result returned

### 4.6 — postgres writer (pgx) [25m] [→4.5,4.1]
- writes Finding row; updates Capture status to `succeeded`
- single transaction; uses `INSERT ... ON CONFLICT DO NOTHING` for idempotency belt-and-suspenders
- DoD: integration test: SQS message → finding visible in `SELECT * FROM findings`

### 4.7 — derived artifact write to S3 [20m] [→4.5,3.3]
- after inference, write derived JSON (and any derived image) to `throne-derived-<env>`
- key format: `findings/<capture_id>.json`
- DoD: object exists in derived bucket after successful processing

### 4.8 — failure handling [35m] [→4.6]
- transient (timeout, connection error): return error without ack, message redelivered after visibility timeout
- permanent (invalid payload, model rejection): ack message, write `failed` Finding with reason
- DoD: inject a corrupt payload → row marked `failed`, message acked, no retry; inject inference timeout → message redelivered 3× then DLQ

### 4.9 — trace propagation through SQS [25m] [→4.5,3.6]
- ingest writes traceparent into SQS MessageAttributes
- processor extracts on receive, continues span
- DoD: single tempo trace shows ingest span → SQS dwell → processor span → inference span, all linked

### 4.10 — integration test (full pipeline) [30m] [→4.9]
- docker-compose up; send 100 captures via grpcurl; assert 100 findings in postgres within 60s
- DoD: test passes; trace visible end-to-end for sampled requests

---

## phase 5 — api + load generator

### 5.1 — api service skeleton [25m] [→1.6,4.1]
- `cmd/throne-api/main.go` — http server, structured logging, OTel, healthcheck
- DoD: starts, `/healthz` returns 200

### 5.2 — JWT validation middleware [30m] [→5.1]
- HS256 in PoC with shared secret in SSM; RS256-ready code path
- claims: user_id, scopes
- DoD: request without token → 401; request with bad sig → 401; valid token → 200 with user_id in context

### 5.3 — capture/finding read endpoints [40m] [→5.2,4.6]
- `GET /v1/users/{id}/captures` (paginated, cursor-based)
- `GET /v1/captures/{id}` (single, authorized by user_id match)
- `GET /v1/findings/{id}` (same authz)
- DoD: integration test asserts pagination, authz boundaries, 404s

### 5.4 — signed-URL issuance [25m] [→5.3,3.3]
- `POST /v1/captures/{id}/image-url` returns S3 presigned URL valid 5 min
- authz: user must own the capture
- DoD: returned URL downloads the original from S3; expires after 5 min

### 5.5 — load generator skeleton [35m] [→3.8]
- `cmd/throne-load/main.go` with flags: `--devices N`, `--rate-per-device`, `--duration`, `--target`
- spawns one goroutine per device, ticker per rate
- emits its own prometheus metrics on :9091
- DoD: `make load -- --devices 10 --rate-per-device 1/s --duration 10s` sends ~100 captures

### 5.6 — load gen device sim with mTLS [25m] [→5.5]
- generates an in-memory device cert chain signed by a dev CA committed in the repo
- DoD: each simulated device authenticates with a distinct cert; ingest accepts them

### 5.7 — load gen synthetic payloads [25m] [→5.6]
- random bytes payload of configurable size (default 5 MB), realistic distribution (1–10 MB lognormal)
- metadata JSON with plausible device telemetry
- DoD: distribution histogram matches expected lognormal within 10%

### 5.8 — nightly load CI workflow [20m] [→5.7]
- workflow that spins up docker-compose, runs 5-min load, asserts SLOs hold
- defaults to manual-trigger for PoC (cost guard)
- DoD: workflow runs end-to-end when triggered manually; artifacts (metrics dump) uploaded

---

## phase 6 — observability + SLO

### 6.1 — grafana dashboard: overview [40m] [→4.10,5.7]
- RED metrics panel per service (request rate, error rate, p50/p99 latency)
- imported as `infra/modules/observability/dashboards/overview.json`
- DoD: dashboard renders against real metrics from a `make load` run; no empty panels

### 6.2 — grafana dashboard: pipeline health [40m] [→6.1]
- queue depth, queue age, DLQ depth per queue; processor throughput; inference inflight
- DoD: chaos run (kill processor) visibly impacts the dashboard

### 6.3 — grafana dashboard: cost (stub for now) [25m] [→6.1]
- panels for bytes/captures/users; cost formulas stubbed with constants
- variables for DAU, captures/user/day
- DoD: dashboard renders; numbers correct for a synthetic 1-min load test (will be wired to real pricing in phase 9)

### 6.4 — alert rules [35m] [→6.2]
- ingest error rate > 1% (5m window), queue age > 60s, DLQ depth > 0, inference p99 > 30s, postgres conns > 80%
- routes to slack webhook (PoC) or PagerDuty (later)
- DoD: each alert fires when its condition is forced; resolves when condition clears

### 6.5 — SLO doc [35m] [→6.4]
- `docs/slo.md` documenting each SLO, error budget calc, burn-rate alerts (Google SRE workbook formulas: 2% in 1h fast, 10% in 6h slow)
- DoD: doc rendered, links from arc42 §10.2 work

### 6.6 — runbooks (top 5 alerts) [40m] [→6.4]
- one markdown per alert in `docs/runbooks/`: symptom, dashboard link, first checks, mitigation, root-cause path
- DoD: each runbook ≤1 page; on-call template applied uniformly

### 6.7 — chaos script [30m] [→6.4]
- `scripts/chaos.sh` with subcommands: `kill-processor`, `saturate-queue N`, `lag-inference MS`, `kill-postgres`, `restore`
- DoD: each command produces a visible impact on dashboards; `restore` returns system to clean state

---

## phase 7 — pricing API wrapper

### 7.1 — service skeleton + cache file [25m] [→3.1]
- `cmd/throne-cost-api/main.go` — http server on :9000
- in-memory cache + local file persistence (`./.cache/pricing.json`)
- 24h TTL with conditional refresh
- DoD: starts, `/healthz` 200, cache file created on first request

### 7.2 — AWS Pricing API client [40m] [→7.1]
- uses aws-sdk-go-v2/service/pricing (region us-east-1 hardcoded for API endpoint; queries any region)
- functions: `GetFargatePricing(region) (vcpuPerHr, gbPerHr, error)`, `GetS3StoragePricing(region, class)`, `GetSQSPricing(region)`, `GetRDSInstancePricing(region, class)`, `GetDataTransferPricing(direction)`
- aggressive caching to avoid Pricing API rate limits
- DoD: each function returns plausible values matching arc42 §6.6 ballpark; tested against real Pricing API once

### 7.3 — HTTP endpoints [25m] [→7.2]
- `GET /v1/pricing/fargate?region=...`, `GET /v1/pricing/s3?region=...&class=...`, etc.
- CORS configured for `localhost:5173` (vite dev) and prod web origin
- DoD: curl returns valid JSON for each endpoint; CORS headers correct

### 7.4 — bundled snapshot fallback [20m] [→7.2]
- `internal/cost/pricing_snapshot.go` with hardcoded values from a known-good fetch
- used when Pricing API unreachable or rate-limited
- DoD: disconnect network, restart service; endpoints still return data with `"source": "snapshot"`

### 7.5 — `make refresh-pricing` target [15m] [→7.4]
- runs the service in one-shot mode that hits the live Pricing API, writes the snapshot file
- DoD: target runs; produces a snapshot file checked into git with date comment

### 7.6 — integration test [20m] [→7.5]
- uses snapshot path (not live API)
- asserts each endpoint returns expected shape, validates pricing math (e.g., Fargate vCPU $0.04048/hr in us-east-1)
- DoD: test runs in CI without AWS credentials

---

## phase 8 — calculator UI

### 8.1 — vite + react + tailwind + shadcn setup [25m] [→7.3]
- `web/cost/` initialized; tailwind config; shadcn/ui installed with sensible component set (button, card, input, slider, select)
- DoD: `pnpm dev` serves on :5173; "hello world" page renders

### 8.2 — cost model (pure TS) [50m] [→8.1]
- `web/cost/src/model/cost-model.ts` — pure functions taking inputs returning breakdown
- exports: `computeMonthlyCost(inputs, pricing) → CostBreakdown`
- handles tiering math (storage accumulation over 24 months with class transitions)
- DoD: unit tested with vitest; 5 boundary cases pass (300 DAU clean, 100k DAU with tiering, etc.)

### 8.3 — pricing fetch + state management [25m] [→8.2,7.3]
- fetches each pricing endpoint on app load, caches in React Query (or simple useState)
- loading + error states
- DoD: mock pricing API in tests; UI handles error state gracefully

### 8.4 — InputPanel component [30m] [→8.3]
- 12 inputs from arc42 §10.3; sliders for ranges, inputs for numbers, selects for enums
- form state managed via react-hook-form or simple useState
- DoD: every change recomputes the model immediately (no submit button)

### 8.5 — CostBreakdown component (stacked bar) [25m] [→8.4]
- uses recharts; stacked bar of monthly cost by category
- DoD: renders; legend correct; tooltip shows $ + %

### 8.6 — PerUserDisplay component [15m] [→8.4]
- big number; color band based on threshold (green ≤$6, amber ≤$8, red >$8)
- DoD: changing DAU updates display; color transitions correctly

### 8.7 — ProjectionChart component [30m] [→8.4]
- 24-month stacked area; accounts for storage accumulation crossing tier boundaries
- DoD: visually matches hand-calculated values for 3 sample scenarios

### 8.8 — ScenarioToggles component [25m] [→8.5,8.6,8.7]
- with/without tiering, Spot, compression — each toggle recomputes; deltas displayed
- DoD: toggling each shows a meaningful delta (storage-tiering toggle dominates)

### 8.9 — PDF export [30m] [→8.8]
- `html2canvas` + `jspdf` capture the dashboard layout into a one-page PDF
- DoD: clicking Export downloads `throne-cost-<timestamp>.pdf`; opens and looks presentable

### 8.10 — UI tests + a11y pass [20m] [→8.9]
- a11y: keyboard-nav all controls, contrast checked
- e2e: playwright script: load → change DAU to 100k → assert color change → toggle tiering → assert delta
- DoD: tests pass in CI

---

## phase 9 — live cost dashboard wiring

### 9.1 — metric emitters for cost-relevant signals [30m] [→6.3]
- ensure ingest emits `throne_storage_bytes_written_total{bucket,class}` per write
- processor emits `throne_captures_processed_total{status}`
- DoD: grafana shows these metrics with expected rates under load

### 9.2 — grafana dashboard panel: current burn rate [35m] [→9.1,6.3]
- panel computes current $/hr from rate(bytes) × pricing constants (templated)
- DoD: panel matches calculator output within 10% for the same configured DAU

### 9.3 — DAU template variable [15m] [→9.2]
- grafana template variable for DAU; all cost panels parameterized by it
- DoD: changing DAU in dropdown updates all panels

### 9.4 — projection panel [25m] [→9.3]
- 24-month projection mirroring the calculator's chart, embedded in grafana
- DoD: matches calculator within 15% (small differences acceptable due to grafana arithmetic limits)

### 9.5 — cost runbook [20m] [→9.4]
- `docs/cost-runbook.md` — how to read the dashboard, what each lever does, escalation rules
- DoD: someone unfamiliar with the system can use it to diagnose "why did costs jump"

---

## phase 10 — demo polish

### 10.1 — top-level README rewrite [40m] [→6.7,8.10,9.5]
- 30s elevator, 60s demo path, architecture diagram, "evaluate me from this repo" pointer list
- explicit "what's not real" section (stub model, no real PHI, sample data)
- DoD: a stranger can clone + `make demo` and have something running in <5 min

### 10.2 — `make demo` orchestration [25m] [→10.1]
- runs `make dev && make migrate && make seed && (make load -- --duration 30s &) && open dashboards`
- DoD: from a clean repo, `make demo` reaches working state in <90s

### 10.3 — demo script [25m] [→10.2]
- `docs/demo-script.md` with timestamped narration for a 5-minute screen-share
- DoD: someone reading the script can deliver the demo in one take

### 10.4 — one-pager PDF [35m] [→10.3]
- single-page summary suitable for emailing recruiter pre-interview
- includes: arch diagram, key metrics, cost-curve teaser, github link
- DoD: PDF exists, looks presentable, prints cleanly

### 10.5 — ADR file split [30m] [→10.1]
- each arc42 §9 ADR broken into `docs/adr/000X-<slug>.md`
- DoD: index `docs/adr/README.md` links all ADRs; each is self-contained

### 10.6 — TODO sweep [20m] [→10.5]
- grep all `TODO`s, resolve or tag `// TODO(post-poc):` with rationale
- DoD: `grep -r "TODO" --include="*.go" | grep -v "post-poc"` returns empty

---

## critical path

minimum wall-clock with infinite parallelism is the longest chain of strict dependencies. mostly serial because each phase depends on the prior, but within phases there's parallelism worth exploiting.

```
0.1 → 0.2 → 0.5 → 1.1 → 1.2 → 1.5 → 1.6 → 1.7
                                      ↓
                                    2.1 → 2.2 → 2.5 → 2.6 → 2.7 → 2.8 → 2.9
                                                                          ↓
                                                                   3.1 → 3.2 → 3.5 → 3.7 → 3.8 → 3.9
                                                                                                 ↓
                                                                                    4.1 → 4.2 → 4.5 → 4.6 → 4.8 → 4.10
                                                                                                                  ↓
                                                                                                          5.1 → 5.2 → 5.3 → 5.7
                                                                                                                          ↓
                                                                                                                  6.1 → 6.2 → 6.4 → 6.7
                                                                                                                                   ↓
                                                                                                                          7.1 → 7.2 → 7.3 → 7.5
                                                                                                                                          ↓
                                                                                                                                  8.1 → 8.2 → 8.4 → 8.8 → 8.10
                                                                                                                                                          ↓
                                                                                                                                                  9.1 → 9.2 → 9.4
                                                                                                                                                                  ↓
                                                                                                                                                          10.1 → 10.2 → 10.4
```

critical path estimate (50%-confidence): ~14 hours
critical path estimate (80%-confidence, ×1.4): ~20 hours

matches the phased-plan estimate. good sanity check.

## parallelization opportunities

- **phase 2 modules:** 2.2, 2.3, 2.4 can be developed in parallel after 2.1 (one engineer, but if pairing or two devs, big win)
- **phase 3 + 4 setup:** 3.5 (mTLS), 3.6 (OTel) and 4.1 (postgres migrations) can run in parallel with the main flow
- **phase 6 dashboards:** 6.1, 6.2, 6.3 independent of each other once 6.1 establishes the data source pattern
- **phase 7 + phase 5 endgame:** 7.x can start as soon as phase 5.1 is done — does not block on phase 6
- **phase 8 components:** 8.5, 8.6, 8.7 can be built in parallel after 8.4
- **phase 10:** 10.4 (one-pager) and 10.5 (ADR split) independent of everything else once 10.1 is done

a sole-developer workflow can't exploit most of this. but it tells you which atoms to defer-without-blocking if you hit a snag.

## risk-weighted critical atoms

these are the atoms most likely to slip and block downstream work. allocate buffer accordingly:

| atom | risk | mitigation |
|---|---|---|
| 2.2 (network module) | VPC + endpoint config is finicky; one typo and nothing else works | start with a working reference (aws-samples/terraform-aws-vpc-module); modify, don't write from scratch |
| 3.5 (mTLS) | cert chain in dev is fiddly | use `mkcert` or generate a one-time CA committed to repo as `dev-only/ca/` |
| 4.9 (trace propagation through SQS) | not built into OTel SDK; custom code; easy to get wrong | reference impl: opentelemetry-go-contrib has propagators for sqs; copy patterns |
| 7.2 (Pricing API client) | Pricing API responses are deeply nested; parsing is error-prone | start by dumping a single response to disk, parse offline first |
| 9.2 (grafana arithmetic for $/hr) | grafana transforms are limited; might need a recording-rule in prometheus | fallback: do the math in a small sidecar that emits a pre-computed metric |

## ground rules

1. **no atom takes >2 hours.** if you hit 2 hours and aren't done, the atom is wrong. split it on the fly.
2. **commit at each atom boundary.** the repo should remain in a working state. if you can't, your atom boundary was wrong.
3. **fail closed.** an atom that "almost works" is incomplete. defer rather than fake-complete.
4. **defer aggressively.** anything that doesn't block demo-day deliverables (e.g., 5.8 nightly CI, 6.6 runbooks beyond top-2, 10.4 polish) is droppable if you're behind. mark dropped atoms in the README's "what's not built" section.
