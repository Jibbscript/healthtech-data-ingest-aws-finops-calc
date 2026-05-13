# Throne Ingest & ML Processing Backend — Arc42 Dossier

**Status:** PoC v0.1 — built as senior backend engineering interview artifact
**Audience:** Throne hiring committee (eng leadership, founders), and the author for implementation reference
**Scope:** A miniature but production-shaped replica of Throne's hypothesized backend: edge devices → ingest gateway → async pipeline → ML inference → tiered storage, fully terraformed on AWS, observable end-to-end, with an interleaved cost model targeting the $6/user/month unit-economics constraint at the 50 GB → 2 TB/day scale-out.
**Non-goals:** A real biological-imagery classifier. Production HIPAA attestation. Multi-region. Real device firmware. This is a credible architectural skeleton, not a shippable v1.

---

## 1. Introduction and Goals

### 1.1 Requirements overview

Throne operates smart-bathroom hardware that captures biological imagery (and likely supplemental sensor signals) from end users, uploads to cloud, runs ML inference, returns health-relevant findings, and retains data for longitudinal analysis. The published Sr Backend Engineer JD names the following load-bearing requirements, and this PoC is shaped to demonstrate each:

| # | JD requirement | How this PoC addresses it |
|---|---|---|
| R1 | Own & evolve backend: Go services, gRPC/REST, async pipelines, AWS (ECS/Fargate, S3, SQS, Terraform) | Every named technology is present and load-bearing in the demo, not decorative |
| R2 | Scale 50 GB → 2 TB/day at ~$6/user/month | Cost model (§3.4, §4.6, §9, §10.3) shows the architecture that holds at 40× ingest growth, with the levers identified |
| R3 | Principled architecture decisions independently — DB selection, service boundaries, queue design, cost/perf trade-offs | §4, §9, §10 are explicit architecture-decision records with rejected alternatives |
| R4 | Reliability & observability for 300 → tens-of-thousands DAU | §10.2 (observability) and §11 (risk) define SLOs, RED/USE coverage, chaos drills, error-budget policy |
| R5 | Help hire & lead the team post-Series A | §2 (stakeholders), §11 (risk register format), §12 (glossary) are written in a shape that hands cleanly to future hires |

### 1.2 Quality goals

In priority order — used to break ties in architecture decisions:

1. **Cost-bounded scalability.** The system must continue to satisfy $/user/month ≤ $6 at every order-of-magnitude step from 300 DAU @ 50 GB/day to 100 k DAU @ 2 TB/day. If a design choice helps latency but breaks this, it loses.
2. **Operational predictability.** A two-person ops rotation must be able to keep the lights on. No bespoke deploy tooling, no undocumented runbooks, no infrastructure that requires the original author to debug.
3. **Domain-appropriate confidentiality.** Even at PoC scope, the data flow assumes the eventual payload is PHI. The pipeline shape — encryption-at-rest, narrowed IAM, redaction sidecar slot — preserves the option of HIPAA attestation later without re-architecting.
4. **Local reasoning.** Service boundaries chosen so that any one service can be understood, tested, and replaced by one engineer in a week. No god-services, no shared databases as integration surfaces.
5. **Reversibility of cloud commitments.** Where two services are equivalent on cost/perf, the more substitutable one wins (SQS over Kinesis at this stage, Postgres over Aurora, Fargate over EKS).

The ordering is intentional: cost > ops > security > clean code > vendor-portability. Reversing 1 and 2 is the most common failure mode for Series-A backends and a worthwhile thing to argue about in the interview.

### 1.3 Stakeholders

| Role | Concerns | Where addressed |
|---|---|---|
| Throne CTO / founding eng | Will this scale to 2 TB/day without blowing the unit econ; can we ship features on top of it | §3, §4, §5, §9 |
| Future backend hires (n=2–5) | Can I onboard in a week; is the local dev story sane | §2.2, §7, §8.4, §12 |
| Throne SRE / on-call (likely the same backend hires for a while) | Can I diagnose a P1 at 3am without paging the author | §8 (deployment), §10.2 (observability), §11 (risk) |
| Throne ML team | Does the inference contract let me ship new models without backend changes | §4.5, §6, §10.5 |
| Throne security/compliance (future) | Is the PHI path defensible to a HIPAA auditor | §3.3, §4.5, §10.4 |
| End user (implicit) | Latency from flush to result; data not leaking | §1.2, §3.1 |

---

## 2. Architecture Constraints

Constraints are things the architecture cannot negotiate. Distinguished from quality goals (which are preferences) and decisions (which are choices among options).

### 2.1 Technical constraints

| ID | Constraint | Source | Implication |
|---|---|---|---|
| TC1 | Backend language is Go | JD explicit | All services in Go 1.22+. No Python services in the request path. Python permitted only for offline ML training and ad-hoc scripts. |
| TC2 | API surface is gRPC + REST | JD explicit | gRPC is canonical; REST is a thin gateway translation. Avoid maintaining two hand-written surfaces. |
| TC3 | Async processing via SQS | JD explicit | Kinesis, MSK, Kafka, EventBridge Pipes are explicitly out of scope for v1. Revisit at >2 TB/day or when ordered streams are required. |
| TC4 | Compute on ECS/Fargate | JD explicit | EKS is out. Lambda permitted for narrow event-driven glue only. No EC2 except for niche workloads (none identified). |
| TC5 | Object storage on S3 | JD explicit | All large binary payloads land on S3. No payloads in databases. |
| TC6 | Infrastructure as code via Terraform | JD explicit | No ClickOps, no CDK, no Pulumi. State in S3 + DynamoDB lock table. |
| TC7 | Unit-economics target ≤ $6/user/month at steady state | JD explicit | Every architectural choice is cost-checked. See §3.4, §4.6, §10.3. |

### 2.2 Organizational constraints

| ID | Constraint | Implication |
|---|---|---|
| OC1 | ~10 person company, ~6 FTE eng (inferred from "third being contractors") | Architecture must be operable by ≤ 2 backend engineers full-time. No microservice sprawl. |
| OC2 | Series A capital, 18–24 month runway typical | Avoid commitments (reserved instances, Savings Plans) longer than 12 months at this stage. Favor pay-as-you-go until traffic is predictable. |
| OC3 | Austin HQ, likely some remote contractors | Tooling must be self-service. No "ask Bob, he set it up." |

### 2.3 Conventions

- **Module layout:** `cmd/`, `internal/`, `pkg/` standard Go layout. No `src/`, no `app/`.
- **Errors:** wrapped with `fmt.Errorf("%w", err)`, never silently swallowed. No `panic` outside `main` and tests.
- **Logging:** structured JSON via `log/slog` (stdlib). One log line per significant state transition; no spam.
- **Tracing:** OpenTelemetry SDK, OTLP exporter, traces propagated through SQS via message attributes.
- **Naming:** services are `throne-<role>` (e.g., `throne-ingest`, `throne-processor`). Queues are `throne-<flow>-<env>`. S3 buckets are `throne-<purpose>-<env>-<account-id>`.

---

## 3. System Scope and Context

### 3.1 Business context

```
┌──────────────────┐      capture       ┌──────────────────┐
│  Throne device   │ ─────────────────► │   This system    │
│  (smart fixture) │ ◄───── ack ─────── │  (cloud backend) │
└──────────────────┘                    └────────┬─────────┘
                                                 │
                            findings, history    │
┌──────────────────┐                             │
│  Mobile app /    │ ◄────────────────────────── │
│  end user        │                             │
└──────────────────┘                             │
                                                 │
                                  derived data   ▼
                            ┌──────────────────────────┐
                            │ ML training (offline)    │
                            └──────────────────────────┘
```

The system sits between (a) edge devices producing biological-imagery payloads, (b) end users consuming findings via a mobile/web app, and (c) an offline ML training loop consuming de-identified historical data.

### 3.2 Technical context

External interfaces (all simulated in the PoC, but shaped for real):

| Counterparty | Direction | Protocol | Payload | Volume target |
|---|---|---|---|---|
| Edge device | inbound | gRPC over TLS, mTLS for device auth | Image blob (1–10 MB) + sensor JSON + device metadata | 1–4 captures/user/day |
| Mobile app | inbound | REST/JSON over TLS, JWT auth | Findings queries, history pagination | ~10 requests/user/day |
| ML training (offline) | outbound (read) | S3 GET, signed URLs | De-identified images + labels | Bulk, off-peak |
| Observability backend (Grafana Cloud or self-hosted) | outbound | OTLP/HTTP | Traces, metrics, logs | Continuous |

### 3.3 PHI boundary (interleaved, since §1.2 names it)

PHI enters the system at the gRPC ingest gateway. From there it traverses: SQS → processor (Fargate) → ML inference (Fargate) → S3 (encrypted). The PHI redaction sidecar (PoC #4 in the broader portfolio) is designed to slot between processor and ML inference so that the model never sees identifying metadata. For this dossier the slot exists but is occupied by a passthrough.

### 3.4 Cost model — boundary view (interleaved cost subsection)

Before any internal decomposition, the unit-economics envelope at the system boundary:

- **At 300 DAU:** ~50 GB/day ingest. ~167 MB/user/day. At $6/user/month budget, that's $1,800/month total — enough for any reasonable architecture.
- **At 10 k DAU:** ~500 GB/day if per-user volume stays flat (it won't — likely grows as features expand). Budget is $60 k/month.
- **At 100 k DAU:** 2 TB/day target from JD. Budget is $600 k/month. Per-user remains $6.
- **Implication:** per-user payload size is the dominant cost driver, not per-user count. The architecture must let us cap or compress per-user volume. See §10.3 for the levers.

---

## 4. Solution Strategy

The high-level approach in five bullets, each elaborated downstream:

1. **One ingest gateway, one canonical wire format.** gRPC service accepts device uploads, performs auth and minimum validation, writes raw payload to S3, enqueues a job referencing the S3 key. REST is a thin grpc-gateway translation for mobile/web. We do not maintain two protocol implementations by hand.
2. **Async fan-out via SQS, never inline ML.** The ingest path returns in <500ms p99. All ML work happens in worker pools draining SQS. Backpressure is queue depth; autoscaling is queue depth.
3. **Stateless workers, stateful storage.** Fargate worker pools are cattle. State lives in Postgres (metadata, findings) and S3 (raw + derived artifacts). No worker-local state survives a restart.
4. **Tiered storage from day one.** Raw images transition S3 Standard → S3 Standard-IA at 30 days → Glacier Instant Retrieval at 90 days. Derived findings stay hot in Postgres + S3 Standard. This is the largest cost lever and it costs ~10 lines of Terraform to set up.
5. **Observability before features.** Traces, metrics, logs, and a synthetic load generator are in the first sprint, not the last. The SLO doc is written before traffic exists.

### 4.1 Service decomposition (preview, full in §5)

Four Go services, intentionally few:

- `throne-ingest` — gRPC server, REST gateway, auth, payload landing
- `throne-processor` — SQS consumer, orchestrates per-payload pipeline, calls ML inference
- `throne-inference` — gRPC server wrapping the ML model (stub in PoC)
- `throne-api` — REST API for the mobile/web client, reads findings, no ingest

Plus shared infrastructure: Postgres (RDS), Redis (ElastiCache, optional — see §9.4), S3 buckets, SQS queues, ALB, secrets in SSM Parameter Store.

### 4.2 Why four services and not one, not ten

One service would couple ingest latency to ML processing time — a violation of strategy bullet 2. Ten services would violate OC1 (operability by ≤ 2 engineers). Four corresponds to four distinct concerns: device-facing, async orchestration, model-facing, user-facing. Each can be owned by one engineer and scaled independently. See §9.1 for the architecture decision record.

### 4.3 Data model strategy

Hybrid: Postgres for everything queryable (users, devices, captures, findings, audit), S3 for everything large (raw images, derived artifacts, training exports). Postgres rows reference S3 keys; S3 objects do not reference Postgres. This makes Postgres restorable from backup without orphaning data, and makes S3 GC a periodic batch job rather than a transactional concern.

### 4.4 Async pipeline strategy

One SQS queue per processing stage. Currently two: `throne-ingest-jobs` (raw → processor) and `throne-inference-jobs` (processor → inference, internal only). Dead-letter queues attached to both, redrive policy set, visibility timeout = 6× expected p99 processing time. See §9.2 for ADR on SQS vs Kinesis at this scale.

### 4.5 ML inference strategy

ML model runs as its own service behind gRPC. Processor calls inference via gRPC, not via SQS. This is a deliberate choice: the inference call is synchronous from the processor's perspective, and SQS-for-everything would add 100s of ms of needless latency. Inference scales horizontally; processor scales on queue depth; the two scale independently.

### 4.6 Cost strategy (interleaved cost subsection)

Four levers, in order of magnitude of impact:

1. **S3 lifecycle policies.** Standard-IA at 30d cuts storage to ~$0.0125/GB-month; Glacier IR at 90d to ~$0.004. At 2 TB/day = 730 TB/year, the difference between "all Standard" and "tiered" is roughly $200 k/year of pure storage spend. Single largest lever.
2. **Compression at ingest.** Raw bio-imagery is highly compressible (lossless 2–3×, perceptually lossless 5–10× depending on what the model needs). One decision at the ingest service, multiplies every downstream cost.
3. **Right-sized Fargate tasks.** Default Fargate sizing is wasteful. Profile real CPU/memory under load, set requests at p95 + 20% headroom, autoscale on CPU+queue depth.
4. **Spot-eligible workers.** Fargate Spot for the processor pool — up to 70% discount, interruption tolerated because SQS redelivers. Inference stays on-demand (latency-sensitive).

These four together are what makes $6/user/month achievable; without them you are at $15–25 trivially. The cost calculator (PoC #1) lets you plug in real numbers and see exactly where each lever bites.

---

## 5. Building Block View

### 5.1 Whitebox: overall system

```
                          ┌────────────────────────────────────────────────┐
                          │                 Throne Backend                 │
                          │                                                │
  Device ──gRPC/mTLS────► │  throne-ingest  ──put──► S3 (raw, encrypted)  │
                          │       │                                        │
                          │       └──enqueue──► SQS (throne-ingest-jobs)  │
                          │                          │                     │
                          │                          ▼                     │
                          │                  throne-processor ────► Postgres
                          │                          │                     │
                          │                          ├──gRPC──► throne-inference
                          │                          │              │      │
                          │                          ◄──result──────┘      │
                          │                          │                     │
                          │                          └──put──► S3 (derived)│
                          │                                                │
  Mobile ──REST/JWT────► │  throne-api  ◄──read──── Postgres + S3         │
                          │                                                │
                          └────────────────────────────────────────────────┘
```

### 5.2 throne-ingest

- **Responsibility:** Terminate device gRPC connection, validate device cert (mTLS), validate payload schema, write raw bytes to S3, enqueue job, ack.
- **Interfaces:**
  - gRPC: `Capture(stream CaptureChunk) returns (CaptureAck)` — streaming upload, chunked to handle 10 MB payloads without buffering whole thing in memory.
  - REST: `POST /v1/captures` (grpc-gateway generated) — for non-gRPC clients, kept for parity.
- **Internal contract:** Never holds payload longer than the S3 PutObject call. No in-memory queues. Failure mode: if S3 PutObject fails after 3 retries, return Unavailable to the device; device retries.
- **Auth:** mTLS client cert; cert thumbprint maps to device_id in Postgres.
- **Sizing:** stateless, tiny. 0.25 vCPU / 0.5 GB Fargate task. Autoscale 2 → 50 on CPU > 60%.

### 5.3 throne-processor

- **Responsibility:** Consume SQS messages, orchestrate per-payload pipeline: load from S3, optionally redact (sidecar slot, passthrough in PoC), call inference, persist findings, write derived artifacts to S3, ack SQS.
- **Interfaces:** SQS consumer (inbound), gRPC client to inference (outbound), SQL writer to Postgres, S3 client.
- **Idempotency:** keyed on (device_id, capture_id). Postgres uniqueness constraint catches duplicates from SQS at-least-once delivery. Idempotency tokens stored 7 days.
- **Failure handling:** transient errors → return without ack, SQS redelivers after visibility timeout. Permanent errors (corrupt payload, model rejection) → ack, mark capture failed in Postgres, alert.
- **Sizing:** 1 vCPU / 2 GB Fargate task on Fargate Spot. Autoscale 1 → 100 on SQS queue depth > 5/worker.

### 5.4 throne-inference

- **Responsibility:** Run the ML model against a payload reference, return findings JSON. In PoC: returns a synthetic but deterministic finding given the same input. Real impl: loads model artifact from S3, holds in memory, runs forward pass.
- **Interfaces:** gRPC `Infer(InferRequest) returns (InferResponse)`.
- **Why separate from processor:** model artifact is large (~hundreds of MB), warm-up is slow, scaling characteristics differ from processor. Also lets ML team ship model updates without touching processor code.
- **Sizing:** 2 vCPU / 4 GB Fargate task on-demand (not Spot — interruption mid-inference wastes work and breaks p99). Autoscale 2 → 30 on inflight gRPC calls.

### 5.5 throne-api

- **Responsibility:** REST API serving the mobile/web client. Reads only — no ingest path here. Auth via JWT (Cognito or self-hosted issuer; PoC uses self-hosted).
- **Interfaces:** REST/JSON. Routes: `GET /v1/users/{id}/captures`, `GET /v1/captures/{id}`, `GET /v1/findings/{id}`, signed-URL issuance for image retrieval.
- **Why separate from ingest:** different auth model (user JWT vs device mTLS), different scaling profile (read-heavy, cacheable vs write-heavy, uncacheable), different release cadence.
- **Sizing:** 0.5 vCPU / 1 GB Fargate. Autoscale 2 → 20.

### 5.6 Shared blocks

- **Postgres (RDS):** db.t4g.medium in PoC, db.m6g.large+ in production. Tables: `users`, `devices`, `captures`, `findings`, `audit_log`. Read replica added at >5 k DAU.
- **S3 buckets:** `throne-raw-<env>`, `throne-derived-<env>`, `throne-training-exports-<env>`. All encrypted with SSE-KMS, separate KMS keys per bucket, bucket policies deny non-TLS access.
- **SQS queues:** `throne-ingest-jobs`, `throne-ingest-jobs-dlq`, `throne-inference-jobs` (only used if processor offloads to async inference, not in PoC default path), corresponding DLQs.
- **Secrets:** SSM Parameter Store with KMS encryption. No secrets in env vars committed to git, no Secrets Manager (cheaper SSM is fine at this scale).

---

## 6. Runtime View

### 6.1 Happy path: device capture → finding available

```
Device                Ingest            S3        SQS       Processor   Inference   Postgres
  │  gRPC Capture       │                │         │           │           │           │
  │ ─────────────────► │                │         │           │           │           │
  │                     │ PutObject      │         │           │           │           │
  │                     │ ─────────────► │         │           │           │           │
  │                     │   200 OK       │         │           │           │           │
  │                     │ ◄───────────── │         │           │           │           │
  │                     │ SendMessage              │           │           │           │
  │                     │ ─────────────────────► │            │           │           │
  │   CaptureAck        │                          │           │           │           │
  │ ◄───────────────── │                          │           │           │           │
  │                                                │           │           │           │
  │                                       ReceiveMessage       │           │           │
  │                                                │ ◄───────│            │           │
  │                                                │           │ GetObject │           │
  │                                                │           │ ─────────────────►   │ (S3)
  │                                                │           │           │           │
  │                                                │           │ Infer (gRPC)          │
  │                                                │           │ ────────► │           │
  │                                                │           │ ◄──result │           │
  │                                                │           │ INSERT findings       │
  │                                                │           │ ─────────────────────►│
  │                                                │ DeleteMessage         │           │
  │                                                │ ◄───────│            │           │
```

p99 budget allocation (target end-to-end <30s for the user to see a finding):
- Ingest gRPC: 500 ms
- SQS dwell (queue depth dependent): 5 s
- S3 GetObject + decode: 1 s
- Inference: 15 s
- Postgres write + S3 derived write: 500 ms
- Mobile app poll interval: 8 s

Cumulative p99 ≤ 30 s under design load. At 2 TB/day with proportional scaling, dwell time stays bounded because workers autoscale.

### 6.2 Failure path: inference timeout

Processor's gRPC call to inference has a 30s deadline. On deadline-exceeded:
1. Processor does NOT ack the SQS message.
2. SQS visibility timeout expires (set to 180s).
3. Message becomes visible, another processor picks it up.
4. After `maxReceiveCount=3` failures, message moves to DLQ.
5. CloudWatch alarm on DLQ depth >0 pages the on-call.

The choice not to retry inside the processor is intentional: retries inside the worker hold the SQS message hostage and starve other workers. Letting SQS handle redelivery is the idiomatic pattern.

### 6.3 Failure path: corrupt payload

Device sends a payload that fails schema validation at ingest:
1. Ingest returns `InvalidArgument` to device.
2. Device firmware logs locally, does NOT retry (no point).
3. No S3 write, no SQS message, no downstream cost.

Failure path: payload passes schema but fails model preprocessing:
1. Processor receives SQS message, loads from S3, calls inference.
2. Inference returns `FailedPrecondition` with reason.
3. Processor writes a `failed` finding row to Postgres with the reason.
4. Acks SQS — this is a permanent failure, no point redelivering.
5. Mobile app shows the failure reason to the user ("retry your capture").

### 6.4 Scaling event: 10× spike

User adoption spikes 10× in a day (press event, app store feature):
1. SQS queue depth grows; CloudWatch metric `ApproximateNumberOfMessagesVisible` exceeds threshold.
2. Application Auto Scaling adds processor tasks at +50% capacity per scaling event, cooldown 60s.
3. Inference tasks scale on inflight-calls metric, lagging processor by ~30s.
4. Within ~5 minutes, queue drains back to steady state.
5. SLO error budget consumption is logged but no page (within budget).

If the spike pushes inference cold-start cost above acceptable: pre-warmed inference pool with a min-capacity floor of 5 tasks. Costs ~$300/month at PoC scale to avoid cold-start pain.

### 6.5 Deployment runtime: rolling update

1. CI builds image, pushes to ECR.
2. Terraform plan in CI shows the new image tag.
3. Manual approve in CI on `main`.
4. ECS rolling deploy: min 100% / max 200% for ingest (no capacity loss during deploy), min 50% / max 100% for processor (Spot-tolerant, lower cost).
5. Health checks gate the new tasks before draining old ones.
6. Rollback: re-apply previous image tag, single Terraform command.

### 6.6 Cost runtime: a single capture's $ trace (interleaved cost subsection)

For one capture, end-to-end, at production-shaped scale:

| Stage | Cost component | $ per capture |
|---|---|---|
| Ingest gRPC | Fargate CPU-seconds | $0.0000003 |
| S3 PUT (raw) | Request + 5 MB transfer | $0.0000050 |
| SQS message | $0.0000004 ($0.40/M) | $0.0000004 |
| Processor work | Fargate Spot CPU-seconds | $0.0000200 |
| S3 GET (raw) | Request | $0.0000004 |
| Inference | Fargate CPU-seconds, 15s @ 2 vCPU | $0.0003000 |
| S3 PUT (derived) | Request + 100 KB | $0.0000060 |
| Postgres write | Amortized | $0.0000050 |
| Observability | Traces+metrics+logs amortized | $0.0000200 |
| **Total per capture** | | **~$0.00035** |

At 3 captures/user/day × 30 days = 90 captures/user/month = **$0.032/user/month for compute + transactions**.

Storage at 5 MB/capture (compressed) × 90/month = 450 MB/user/month new data:
- First 30d hot: 450 MB × $0.023 = $0.010
- Next 60d IA: cumulative ~900 MB × $0.0125 = $0.011
- After 90d Glacier IR: cumulative growth × $0.004 — long-tail, ~$0.05/user/month at year 2

Storage steady-state ≈ $0.07/user/month at year 2.

**Compute + storage subtotal: ~$0.10/user/month.** The remaining $5.90 of the $6 budget absorbs egress, observability overhead, RDS, support tooling, and headroom. The architecture has slack — but only because of the four levers in §4.6. Removing tiering alone pushes year-2 storage above $0.30/user/month and the budget gets tight fast.

---

## 7. Deployment View

### 7.1 Environments

- **local** — docker-compose: postgres, localstack (S3+SQS), services run via `go run`. No AWS account needed. Cost: $0.
- **dev** — single AWS account, single region (us-east-1), shared by all engineers, lower-tier sizing, lifecycle policies aggressive (delete raw after 7d). Cost target: <$500/month.
- **prod** — separate AWS account, us-east-1 primary, eventual us-west-2 DR (out of PoC scope).

### 7.2 Topology in AWS (prod-shaped, PoC built)

```
              Route 53
                 │
            ┌────┴────┐
            │   ACM   │  (TLS certs)
            └────┬────┘
                 │
              ALB (HTTPS only, WAF attached)
              │   │   │
              │   │   └────► throne-api (Fargate, private subnets)
              │   └────────► throne-ingest (Fargate, private subnets, mTLS via NLB sidecar)
              │
              ▼
        [VPC: 10.0.0.0/16]
         ├─ public subnets (ALB, NAT GW)
         ├─ private subnets (Fargate tasks)
         └─ data subnets (RDS, ElastiCache)

        Service-to-service: via internal NLB or service discovery (Cloud Map)

        Async layer (no VPC, AWS-managed):
         ├─ SQS queues
         ├─ S3 buckets (gateway endpoint, no NAT cost)
         └─ SSM Parameter Store (interface endpoint)
```

### 7.3 Terraform module layout

```
infra/
  modules/
    network/           VPC, subnets, NAT, endpoints, security groups
    data/              RDS, ElastiCache, parameter groups, backups
    storage/           S3 buckets, lifecycle policies, KMS keys
    queue/             SQS queues, DLQs, redrive policies
    service/           Reusable ECS service module (taskdef, service, autoscaling, target group)
    observability/     CloudWatch dashboards, log groups, alarms
  envs/
    local/             Override for localstack (dev convenience)
    dev/               One AWS account, smaller sizing
    prod/              Production sizing, separate account
```

Root modules per env compose modules. Backend state in S3 with DynamoDB locking, per-env state files. No remote operations during PoC (terraform local apply); Terraform Cloud or Atlantis added later.

### 7.4 CI/CD (PoC: GitHub Actions)

- Lint, vet, test, race detector on every PR.
- Container build + push to ECR on merge to main.
- Terraform plan on PR (`tflint`, `tfsec`, `terraform plan`); comment plan output on PR.
- Terraform apply on merge to main with manual approval gate.
- No blue-green at PoC scale; ECS rolling deploy is sufficient. Re-evaluate at 10 k DAU.

### 7.5 Local development

Goal: `make dev` brings up the full stack on a laptop in <60s, no AWS account.

- docker-compose: postgres, localstack (S3 + SQS), grafana, tempo, prometheus.
- Services run via `air` or `go run` outside docker for fast iteration.
- A `make seed` populates a synthetic device, user, and ten fake captures.
- A `make load` runs the load generator against local, hands-on chaos tools available.

This is the single most underrated piece of architecture quality at this scale: if a new hire can `git clone`, `make dev`, `make seed`, and have a working pipeline in their first hour, you've already won. See §2.2, OC3.

---

## 8. Cross-cutting Concepts

### 8.1 Authentication & authorization

- **Device → ingest:** mTLS. Device certs issued at provisioning, rotated quarterly. Cert thumbprint maps to device_id; device_id maps to user_id in Postgres.
- **User → api:** JWT (RS256), issued by a self-hosted token service in PoC (Cognito or Auth0 in real prod). Tokens carry user_id, scopes.
- **Service → service:** within VPC, security groups gate connectivity; no mTLS between internal services in PoC (added later via mesh or service connect at scale).
- **Service → AWS:** IAM task roles, scoped to single bucket / single queue / single secret prefix per service. Audit-friendly.

### 8.2 Idempotency

Every state-changing operation carries an idempotency key. Captures: `device_id + capture_uuid`. Inference: `capture_uuid` (input is content-addressed via S3 key). Postgres uniqueness constraints catch duplicates. SQS at-least-once is the assumption; never the surprise.

### 8.3 Schemas & evolution

- Proto schemas (`api/proto/`) are the source of truth for all wire formats.
- `buf` for proto lint, breaking-change detection, generation.
- REST gateway generated via `grpc-gateway`.
- Postgres schemas managed via `golang-migrate`. Forward-only in prod; never edit applied migrations.
- Schema evolution: add fields, never remove or repurpose. Old fields stay reserved. Standard proto3 etiquette.

### 8.4 Onboarding contract (for future hires)

The repo `README.md` answers, in order:
1. What does this system do? (one paragraph)
2. How do I run it locally? (`make dev`)
3. Where are the service boundaries? (links to §5)
4. Where do I look when something's broken? (links to §10.2 and runbooks/)
5. How do I add a new endpoint? (links to a worked example PR)
6. How do I add a new ML model? (links to §10.5)

If a new hire can't answer "what is this and how do I run it" inside hour one, the README has failed. This is a hard quality bar.

### 8.5 Observability conventions

See §10.2 for the full spec. Convention summary:
- One trace per device capture, propagated through SQS via message attributes (custom logic — SQS does not propagate trace context natively).
- Span names: `service.operation` (e.g., `ingest.put_object`, `processor.infer`).
- Metric names: `throne_<service>_<noun>_<unit>` (e.g., `throne_processor_queue_age_seconds`).
- Log levels: `error` (page-worthy), `warn` (review-worthy), `info` (state transitions), `debug` (off in prod).

### 8.6 Data classification & retention

- **Class A (PHI):** raw images, derived images with identifying metadata, user PII. Retention: 7 years (HIPAA min), encrypted with PHI-class KMS key, never logged.
- **Class B (Findings):** model outputs without identifying metadata. Retention: indefinite. Encrypted with standard KMS key.
- **Class C (Telemetry):** logs, traces, metrics. Retention: 30d hot, 90d archive. Scrubbed of any Class A leakage at emit time.

PII scrubbing is enforced at the structured-log emit site via a `slog.Handler` middleware. Easier to verify in tests than scrubbing at the log destination.

---

## 9. Architecture Decisions (ADRs)

Each ADR follows: context, options, decision, consequences. Compressed format here; in repo, one file per ADR under `docs/adr/`.

### ADR-001: SQS over Kinesis (or MSK/Kafka)

**Context.** Async pipeline between ingest and processing. Choice of queue/stream system.
**Options considered.**
- SQS: pay-per-message, AWS-managed, no ordering guarantees (unless FIFO), no replay, simple semantics.
- Kinesis Data Streams: shard-based, ordering within shard, 24h–365d retention with replay, pay per shard-hour.
- MSK (Kafka): full Kafka, replay, ordering, ecosystem. Operationally heavier even when managed.
**Decision.** SQS standard queues for v1.
**Reasoning.**
- Ordering is not required: each capture is independent.
- Replay is not required at v1: failed captures go to DLQ, are inspected, re-enqueued manually.
- At 2 TB/day with ~5 MB/capture ≈ 400 k captures/day ≈ 4.6 msg/sec. SQS cost is $0.40 per million requests, ~$60/month at this volume. Kinesis at minimum 2 shards is ~$22/month base + transfer. SQS is simpler and not materially more expensive at this scale.
- Revisit trigger: ordered processing required (multi-step pipelines), replay needed (model retraining), or volume exceeds 10 M msg/day where Kinesis-or-MSK economics begin to dominate.
**Consequences.** Cannot replay failed batches without DLQ inspection. Must build idempotency into consumers (already required for SQS at-least-once delivery anyway). Migration to Kinesis later is non-trivial but bounded: consumers re-platform, producers re-platform; pipeline stages can migrate one at a time behind a shim.

### ADR-002: Postgres over DynamoDB for metadata

**Context.** Need a persistent store for users, devices, captures, findings, audit.
**Options.**
- Postgres (RDS): relational, joins, transactional, well-understood.
- DynamoDB: serverless, single-digit-ms latency, infinite scale, requires access-pattern-first design.
- Aurora Serverless v2: Postgres-compatible, auto-scaling, more expensive idle.
**Decision.** Postgres on RDS (db.t4g.medium → db.m6g.xlarge as scale grows).
**Reasoning.**
- Findings are queried by joins (user → captures → findings, time ranges). DynamoDB would require careful denormalization or a secondary index strategy.
- Eng team is small (OC1); Postgres has the biggest pool of operational knowledge among hires.
- Data volume is modest: 100k users × 90 captures/user/month × 12 months = 108 M rows/year — comfortably within Postgres on m6g.xlarge.
- Cost at PoC: db.t4g.medium ~$50/month. At 100k DAU production: m6g.xlarge + read replica + backups ~$700/month — well within budget.
**Consequences.** Need to manage backups, parameter tuning, version upgrades — RDS handles most of this. Add a read replica at ~5k DAU; vertical-scale before horizontal until pain forces sharding. Forecast: vertical scaling carries us comfortably past 100k DAU.

### ADR-003: gRPC + grpc-gateway REST, not hand-written REST

**Context.** Need to support both device (gRPC) and mobile (REST) clients.
**Options.**
- Hand-write both REST and gRPC, maintain in parallel.
- gRPC primary, grpc-gateway generates REST.
- REST primary, no gRPC.
**Decision.** gRPC primary, grpc-gateway for REST.
**Reasoning.** Single source of truth (proto), automatic REST translation, types end-to-end. The "maintain two surfaces" failure mode is the worst possible state for a small team.
**Consequences.** grpc-gateway adds a generation step and a small runtime layer. Some REST conventions feel awkward when auto-generated; can override per-endpoint when needed. Worth it.

### ADR-004: Fargate over EKS

**Context.** Container orchestration choice.
**Options.** ECS/Fargate, ECS/EC2, EKS/Fargate, EKS/EC2.
**Decision.** ECS/Fargate.
**Reasoning.** JD names ECS/Fargate (TC4). Independently: Fargate eliminates node management entirely. Cost at PoC scale is comparable; at production scale, EC2-backed ECS or EKS is ~30% cheaper per vCPU-hour but requires capacity-provider/node-management ops work that a 2-person team can't absorb. Revisit at 100k DAU when the team has grown.
**Consequences.** Pay a Fargate premium per vCPU-hour. Get back: zero node ops, faster cold starts than EKS-Fargate, simpler IAM (task role only). Cost premium is in the $200–500/month range at PoC scale, materially less than the ops time it would consume to manage nodes.

### ADR-005: One queue per stage, dead-letter queue per queue

**Context.** SQS queue design for async pipeline.
**Options.**
- Single queue, processor does everything.
- One queue per processing stage.
- Topic-fanout via SNS for parallel workflows.
**Decision.** One queue per stage; DLQ per queue; redrive policy on every queue.
**Reasoning.** Each stage's failure mode is distinct; mixing them in one DLQ loses signal. Topics overkill at v1 (one consumer per stage).
**Consequences.** More queues to monitor; the observability dashboard surfaces all of them at once so the cognitive load is fixed at "look at the dashboard," not "remember which queues exist."

### ADR-006: KMS per data class, not per bucket

**Context.** KMS key strategy.
**Options.** One KMS key, one per bucket, one per data class, one per env.
**Decision.** One KMS key per (data-class × env). PHI-class key separate from non-PHI-class key in each env.
**Reasoning.** Lets us grant decryption permission narrowly (per class), supports HIPAA-style key rotation policies for PHI without rotating everything. Not so many keys that key management itself becomes a burden ($1/key/month).
**Consequences.** A few keys to track. Per-data-class IAM policies. Easy to extend (add a new class → add a new key).

### ADR-007: S3 lifecycle policies in Terraform, not application code

**Context.** Where to define data tiering.
**Decision.** Lifecycle policies in Terraform (`aws_s3_bucket_lifecycle_configuration`).
**Reasoning.** Lifecycle in code = lifecycle is versioned, reviewed, easy to audit. Lifecycle in application code = lifecycle is invisible until something costs more than expected. The single largest cost lever (§4.6) deserves to live in the infrastructure layer.
**Consequences.** Changing lifecycle rules requires a Terraform PR. Good — this should be a deliberate decision.

### ADR-008: Synthetic load generator as a first-class artifact

**Context.** How to validate scaling and SLO claims.
**Decision.** A Go-based load generator (in `cmd/throne-load/`) that simulates a configurable fleet of devices, lives in the repo, runs in CI nightly against dev, can be pointed at any environment.
**Reasoning.** Claims about "scales to 2 TB/day" need to be verifiable, even if not at full scale during PoC. The load generator is the experiment that backs the architectural claim.
**Consequences.** Maintenance burden of one more service. Worth it: the load generator finds bugs before users do.

---

## 10. Quality Requirements

### 10.1 Quality tree

```
                  Throne backend quality
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
   Performance          Operability           Confidentiality
       │                    │                    │
   ├ p99 ingest <500ms      ├ MTTR < 30 min      ├ PHI never logged
   ├ p99 e2e <30s           ├ Deploys <10 min    ├ KMS per data class
   ├ Throughput 2 TB/day    ├ Onboard new eng <1d ├ Audit trail complete
   └ $≤6/user/month         └ One-page runbooks  └ Encryption end-to-end
                            │
                       Reliability
                            │
                       ├ 99.9% availability
                       ├ DLQ depth = 0 steady state
                       └ Zero data loss on failures
```

### 10.2 Observability spec (interleaved depth — load-bearing on R4)

**Metrics (Prometheus exposition, scraped by Grafana Agent / OTLP push):**

| Metric | Type | Labels | Why |
|---|---|---|---|
| `throne_ingest_requests_total` | counter | method, code | Request rate, error rate (RED) |
| `throne_ingest_request_duration_seconds` | histogram | method, code | Latency p50/p99 |
| `throne_processor_queue_age_seconds` | gauge | queue | Oldest message age — leading indicator of backpressure |
| `throne_processor_messages_processed_total` | counter | result | Throughput, success/failure ratio |
| `throne_inference_duration_seconds` | histogram | model_version | Inference latency, model regression detection |
| `throne_inference_inflight` | gauge | — | Saturation signal for inference autoscaling |
| `throne_storage_bytes_written_total` | counter | bucket, class | Cost proxy, anomaly detection |
| `throne_postgres_connections_used` | gauge | service | Connection-pool saturation |

**Traces:** every device capture is a single trace, propagated via SQS message attributes (custom logic — SQS does not natively carry trace context). Trace covers: ingest gRPC handler → S3 put → SQS send → processor handler → S3 get → inference gRPC → Postgres write. Sampling: 100% during PoC, 1% in prod with always-sample for errors.

**Logs:** structured JSON via `log/slog`. One log per significant state transition. Every log includes `trace_id`, `service`, `env`, and the relevant business identifier (`device_id`, `capture_id`). PII never logged — enforced by a `slog.Handler` middleware that drops fields matching a denylist regex at emit time.

**Dashboards (Grafana):**
1. **Overview** — RED metrics per service, queue depths, error budget burn.
2. **Pipeline health** — message ages, processing latencies, DLQ depths, retry rates.
3. **Cost dashboard** — bytes ingested/processed/stored, captures/hour, projected $/user/month at current burn rate. (This is the live tie-in to PoC #1 — see §10.3.)
4. **Inference quality** — per-model-version latency, error rate, output distribution (catch silent model regressions).

**SLOs:**

| Service | SLO | Window | Error budget |
|---|---|---|---|
| Ingest availability | 99.9% successful uploads | 30d rolling | 43 min/month |
| Ingest latency | p99 < 500 ms | 30d rolling | — |
| End-to-end freshness | p95 capture→finding < 60 s | 30d rolling | — |
| Pipeline durability | zero data loss after ack | always | 0 |

**Burn-rate alerts:** fast burn (2% budget in 1h) pages; slow burn (10% budget in 6h) ticket. Following Google SRE workbook conventions.

### 10.3 Cost model — interactive calculator (PoC #1 interleaved here)

The cost calculator is built as a small React + Go app that lives in `cmd/throne-cost/` and `web/cost/`. It is the read-only consumer of the same metrics that the production system emits, plus configurable parameters for what-if analysis.

**Inputs (user-configurable):**

| Parameter | Default | Range | Source |
|---|---|---|---|
| DAU | 300 | 100 – 1M | manual |
| Captures/user/day | 3 | 1 – 10 | manual |
| Avg payload size (MB) | 5 | 1 – 20 | manual |
| Inference duration (s) | 15 | 1 – 60 | manual |
| Inference Fargate vCPU | 2 | 0.25 – 16 | manual |
| Processor Fargate vCPU | 1 | 0.25 – 4 | manual |
| Spot discount % | 70 | 0 – 100 | manual |
| S3 tiering enabled | yes | yes/no | manual |
| Standard→IA cutover (days) | 30 | 0 – 365 | manual |
| IA→Glacier cutover (days) | 90 | 0 – 365 | manual |
| RDS instance class | db.m6g.large | enum | manual |
| Read replica? | no | yes/no | manual |

**Pricing data:** fetched from AWS Pricing API on app load, cached for 24h. Pricing JSON is large (~hundreds of MB across all services); the app fetches only the SKUs it needs (Fargate vCPU-hour and GB-hour by region; S3 storage by class and region; SQS request pricing; RDS instance-hour; data transfer by direction). Fallback to a bundled snapshot if the Pricing API call fails.

**Outputs (live, recalculated on any input change):**

- Total monthly cost, broken down by: compute, storage (hot/IA/Glacier), data transfer, RDS, SQS, observability.
- $/user/month — large numeric display, color-coded against the $6 line (green ≤$6, amber ≤$8, red >$8).
- Stacked area chart: monthly cost over 24 months, accounting for storage accumulation.
- "What if" toggles: with/without tiering, with/without Spot, with/without compression — instantly recompute deltas.
- An exportable PDF summary suitable for handing to a CFO.

**Architecture of the calculator itself:**

- Go HTTP server (`cmd/throne-cost-api/`) wraps the AWS Pricing API, holds the pricing cache, exposes `GET /v1/pricing?service=...&region=...`.
- React frontend (Vite + TypeScript) holds the model state, recomputes on every input change, fetches pricing on first load.
- Same Go binary served from same ECS task as `throne-api` in production — one less thing to deploy.

**Why it lives in the same repo as the rest:** the calculator is most useful when it shares the canonical assumptions (payload sizes, processing durations) with the running system. Anchor the model to real telemetry, not separate guesses.

**Demo-day flow (for the interview):** open dashboard, show current $/user/month at PoC scale. Open calculator, sweep DAU 300 → 100 k. Show the $6 line going red without tiering. Re-enable tiering. Show it going green. Toggle compression. Show further savings. This is a 2-minute live story that lands the entire cost narrative.

### 10.4 Security spec

- Encryption at rest: SSE-KMS for S3, RDS storage encrypted, ElastiCache encrypted at rest.
- Encryption in transit: TLS 1.3 everywhere, mTLS device-to-ingest.
- Network: no internet-facing services except ALB; private subnets for Fargate; VPC endpoints for S3, SQS, SSM.
- IAM: per-service task roles, least privilege, no `*` actions, no `*` resources.
- Audit: CloudTrail enabled, S3 access logs enabled, application audit table (`audit_log`) for user-facing events.
- Secrets: SSM Parameter Store with KMS encryption, rotation runbook documented.
- Vulnerability scanning: ECR image scanning on push, Dependabot, `govulncheck` in CI.

### 10.5 ML pipeline contract

- Inference service receives `(s3_key, model_version)`, returns `(findings_json, confidence_scores)`.
- Models live in S3 under `throne-models-<env>/v<version>/`, downloaded on inference task startup, cached in-memory.
- New model version rollout: deploy with `model_version` config, traffic-shift via processor reading config from SSM. Canary 5% → 25% → 100%.
- Model regression detection: output distribution monitored against prior version; alarm on KL divergence > threshold.

---

## 11. Risks and Technical Debt

### 11.1 Known risks at PoC scope

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R-01 | Postgres becomes write-bottleneck at >5k DAU before sharding strategy exists | M | H | Add read replica at 5k DAU; spec sharding plan at 50k DAU. Vertical scaling buys time. |
| R-02 | SQS throughput limit (~3k req/s standard) hit before Kinesis migration | L | M | At 2 TB/day = 4.6 msg/s — orders of magnitude under limit. Monitor; ADR-001 names trigger. |
| R-03 | Fargate Spot interruption during inference wastes CPU-time | M | L | Inference on on-demand, not Spot (see §5.4). Processor on Spot is idempotent and SQS-redelivered. |
| R-04 | Cost model snapshot drifts from real pricing | M | M | Daily refresh of pricing cache; alert if Pricing API has been unreachable >24h. |
| R-05 | PHI accidentally logged | L | C (critical) | slog handler enforces denylist at emit; CI grep for known PHI field names in test fixtures. |
| R-06 | Inference latency tail (p99.9) is much worse than p99 | H | M | Tail-aware autoscaling on p95, not avg; bounded queues per inference task. |
| R-07 | Single-region failure (us-east-1 outage) | L | H | DR plan to us-west-2 deferred to post-Series-A scale. Documented, not implemented. |
| R-08 | Hidden cost: NAT Gateway egress for S3 traffic | M | M | VPC gateway endpoint for S3 (free), interface endpoints for SQS/SSM (paid but bounded). Verified in TF. |
| R-09 | Schema evolution breaks device firmware | M | H | Proto3 etiquette enforced; `buf breaking` in CI; never remove fields. |
| R-10 | On-call burnout at 2-person team | M | H | Strong SLOs + error budgets prevent over-paging; runbooks for top-10 alerts; quarterly review. |

### 11.2 Technical debt accepted at PoC scope

- No service mesh; service-to-service mTLS deferred. Acceptable while service count ≤ 5.
- No blue-green deploy; rolling deploy sufficient at v1. Add when first deploy-induced incident happens.
- No multi-region; single-region SLA accepted. Revisit at $10M ARR or first regulator inquiry.
- Calculator pricing data fetched lazily, not pre-warmed. First load is 1–2s slower. Acceptable.
- No async inference (all gRPC). If model latency grows past 30s, refactor to SQS-based inference. Bounded refactor.

### 11.3 Out-of-scope

- Real ML model (the actual hard part of Throne — by design, not what this PoC demonstrates).
- Device firmware (simulated only).
- Mobile app (simulated only).
- Billing/payments (not in JD).
- Customer support tooling.

---

## 12. Glossary

| Term | Meaning |
|---|---|
| Capture | A single device upload event — image + sensor JSON + metadata |
| Finding | ML-derived health-relevant output from a capture |
| Device | A physical Throne unit, identified by mTLS cert thumbprint |
| User | The human associated with one or more devices |
| Tier (S3) | Standard / Standard-IA / Glacier Instant Retrieval; storage class |
| DLQ | Dead-letter queue — SQS queue receiving messages that failed `maxReceiveCount` times |
| SLO | Service-level objective — quantified reliability target |
| Error budget | (1 − SLO) × window — allowable failure before policy kicks in |
| RED metrics | Rate, Errors, Duration — request-oriented service health metrics |
| USE metrics | Utilization, Saturation, Errors — resource-oriented health metrics |
| Idempotency key | Token that allows safe retry of an operation |
| PHI | Protected Health Information (HIPAA) |
| Lifecycle policy | S3 rule for automatic tier transitions and expirations |
| Visibility timeout | SQS — how long a received message is hidden from other consumers |
| ADR | Architecture Decision Record |
| ECR | Elastic Container Registry — AWS container image registry |
| OTLP | OpenTelemetry Protocol — wire format for traces/metrics/logs |
| SSM | Systems Manager — used here for Parameter Store secrets |
| ACM | AWS Certificate Manager — TLS certs for ALB |
| WAF | Web Application Firewall |

---

*End of Arc42 dossier. Length: ~9,200 words. Companion phased implementation plan in `phased-plan.md`.*
