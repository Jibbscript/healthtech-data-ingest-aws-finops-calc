# arc42 cost-section patch — v0.2

**Replaces** sections §3.4, §4.6, and §6.6 of `arc42-throne-ingest-poc.md` with numbers verified against AWS pricing pages, us-east-1, retrieved May 2026. Conclusions unchanged; magnitudes corrected.

**Change summary vs v0.1:**
- Per-capture cost was ~$0.00035; real is ~$0.00046 (x86) or ~$0.00038 (ARM/Graviton). Driver: inference memory was undercounted.
- Per-user-month at year-2 steady state was ~$0.10; real is ~$0.14.
- Architecture still has ~$5.86 of slack against $6/user/mo budget — the thesis holds.
- New first-class lever surfaced: **ARM/Graviton across the fleet** (~20% on compute).
- New gotcha: **S3 lifecycle policies no longer transition objects <128KB by default** (post Sept 2024) — derived findings affected.

---

## §3.4 — Cost model, boundary view (revised)

Same business-envelope framing as v0.1. Numbers updated:

- **At 300 DAU:** ~50 GB/day ingest; ~167 MB/user/day. $6/user/mo × 300 = **$1,800/mo total budget** — comfortably covers any reasonable architecture, including PoC waste.
- **At 10 k DAU:** ~500 GB/day if per-user volume holds. Budget **$60 k/mo**.
- **At 100 k DAU:** 2 TB/day target. Budget **$600 k/mo**. Per-user $6.
- **Implication:** per-user payload size remains the dominant cost driver. The architecture must cap per-user volume (compression, server-side dedup on identical sensor frames, capture-rate limits). See §10.3 for the levers.

---

## §4.6 — Cost strategy (revised), in descending magnitude

1. **S3 lifecycle policies — Standard → Standard-IA at 30d → Glacier IR at 90d.** At 2 TB/day = 730 TB/year accumulating, the difference between "all Standard" and "fully tiered" is approximately:
   - All Standard at year 2 (1460 TB): $33,580/month for raw alone
   - Tiered (last 30d Standard, next 60d IA, rest Glacier IR): ~$5,400/month
   - **Delta ≈ $28,000/month at year-2 100k-DAU scale.** Single largest lever, full stop.

2. **Compression at ingest.** Throne's payload is sensor + imagery; realistic lossless compression ratio 2-3×, perceptually lossless 5-10× if the model accepts it. A 5× drop multiplies every downstream storage and transfer cost. Worth investing engineering hours in.

3. **ARM/Graviton across the fleet.** ~20% off all Fargate compute on equivalent workloads. New as a called-out lever in v0.2 — was implicit before. Inference, processor, ingest, API all viable on ARM with Go (single binary recompile). At 100k DAU: roughly $1,200/month savings from ARM-only.

4. **Right-sized Fargate tasks.** Default sizing is wasteful. Profile real CPU/mem under load, request at p95 + 20% headroom, autoscale on CPU+queue depth. Magnitude is workload-dependent but typically 20-40% over naive sizing.

5. **Spot for the processor pool.** Fargate Spot ~70% off on-demand. Interruption acceptable: SQS redelivers. **Do not Spot the inference pool** — interruption mid-inference wastes work and breaks p99. At 100k DAU: processor Spot saves ~$600/month vs. on-demand processor.

Stacking levers 1+2+3 is the difference between $0.30/user/month (no tiering, no compression, x86) and $0.14/user/month (fully tiered, compressed, ARM). At 100k DAU that's a $16 k/month spread.

---

## §6.6 — Per-capture cost trace (revised, verified May 2026)

Workload assumptions (unchanged from v0.1):
- 3 captures/user/day, 90 captures/user/month
- Compressed payload: 5 MB/capture
- Inference: 15s on 2 vCPU / 4 GB
- Processor: ~2s on 1 vCPU / 2 GB on Fargate Spot ARM
- Ingest: ~500ms on 0.25 vCPU / 0.5 GB ARM, amortized
- Derived findings: ~100 KB JSON

### per-capture trace, x86 inference

| stage | cost component | $ per capture |
|---|---|---|
| Ingest gRPC | Fargate ARM compute, amortized | $0.0000010 |
| S3 PUT (raw, 5 MB) | request + inbound transfer (free) | $0.0000050 |
| SQS SendMessage | $0.40/M, well under 64KB | $0.0000004 |
| Processor (ARM Spot) | 2s × (1 vCPU + 2 GB) at 50-70% off ARM rates | $0.0000100 |
| S3 GET (raw) | request only; intra-region free via gateway endpoint | $0.0000004 |
| **Inference (x86)** | **15s × (2 vCPU + 4 GB) on-demand** | **$0.0004100** |
| S3 PUT (derived, 100 KB) | request | $0.0000050 |
| Postgres write | amortized RDS m6g.large + storage at 100k DAU | $0.0000050 |
| Observability | CloudWatch logs + Tempo traces, amortized | $0.0000200 |
| **TOTAL per capture** | | **~$0.00046** |

### per-capture trace, ARM inference (alt — model retraining required)

If the ML model can be recompiled/served on ARM/Graviton:

| stage | $ per capture |
|---|---|
| Inference (ARM) | **$0.0003300** |
| (all others unchanged) | $0.0000470 |
| **TOTAL** | **~$0.00038** |

Worth ~$1,400/month at 100k DAU. Conditional on the ML team being willing to ship ARM model artifacts (PyTorch and TF both support; depends on custom ops).

### per-user-month at year-2 steady state, 100k DAU production

- 90 captures × $0.00046 = **$0.041/user/month** for compute + transactions
- Storage tiered, 24-month accumulation: **$0.070/user/month**
- RDS m6g.xlarge + read replica amortized: **$0.007/user/month**
- NAT + ALB + control plane (fixed costs spread over DAU): **$0.001/user/month**
- Egress (mobile app pulls): **$0.001/user/month**
- Observability fixed overhead: **$0.020/user/month**
- **subtotal: ~$0.14/user/month**
- **headroom: ~$5.86/user/month against $6 budget**

That $5.86 absorbs: dev/staging environments, security tooling, monitoring SaaS if you use one, support tooling, billing, fraud detection, ML training compute (the actually expensive part of an ML company), and future feature compute.

### per-user-month at NO-TIERING scenario (counterfactual)

If you skip lifecycle policies and keep everything in S3 Standard:
- year-1: ~$0.16/user/mo
- year-2: ~$0.30/user/mo
- year-3: ~$0.43/user/mo
- year-5: ~$0.70/user/mo

Storage cost grows linearly with time-in-service per user. **Without tiering, by year 5 you're spending 5× as much on storage per user as on everything else combined.** This is the lever that matters.

### per-user-month at NO-COMPRESSION scenario (counterfactual)

If you skip compression and store raw 25 MB blobs:
- year-2 storage: ~$0.35/user/mo (vs. $0.07 compressed)
- Still survivable against $6 budget at year 2
- At year 5: ~$1.75/user/mo — starting to eat real budget

### per-user-month at NO-SPOT scenario (counterfactual)

If you put processor on on-demand instead of Spot:
- per-capture processor: $0.00003 instead of $0.00001
- per-user-month delta: +$0.0018
- **trivial.** Spot is the smallest of the four levers in raw $ terms.

---

## §6.6 addendum — known gotchas baked into the model

These are pricing realities that affect the math in non-obvious ways:

1. **128KB lifecycle exclusion (post Sept 2024).** S3 lifecycle rules no longer transition objects under 128 KB by default. Our derived findings (~100 KB) are affected. Options:
   - (a) Accept that derived findings stay in S3 Standard. Math holds; they're tiny.
   - (b) Pack derived findings into larger objects (e.g., per-user-per-month rollup) so they exceed the threshold. Adds compaction job complexity.
   - (c) Explicitly opt in to small-object transitions via the new lifecycle config flag (`Filter.ObjectSizeGreaterThan=0`). Possible but tracks more lifecycle requests.
   - **Recommended:** (a) for v1. Cost impact negligible.

2. **Lifecycle transition requests aren't free.** Transitions to IA cost $0.01/1000 ($10/M); to Glacier IR roughly $0.02/1000 ($20/M). At 90 transitions/user/month + 90 more later: ~$0.0027/user/month in transition fees. Already included in the $0.14 number.

3. **VPC gateway endpoints for S3 and DynamoDB are free.** Interface endpoints (SQS, SSM, ECR, etc.) cost $0.01/hr per endpoint per AZ + $0.01/GB processed. At ~3 AZs × 5 endpoints × $0.01/hr × 730 hr = ~$110/month fixed. Per-user at 100k DAU: $0.0011/user/month — trivial. Already baked into the "fixed costs" bucket above.

4. **CloudWatch Logs ingestion ($0.50/GB) is the silent killer of observability budgets.** If you log naively (full request body, every field), 100k DAU can produce hundreds of GB/day of logs. Our budget of $0.02/user/month assumes ~5 MB/user/month of structured logs — disciplined, but achievable. If you blow past, the cost line moves fast.

5. **NAT gateway fixed cost is $32.85/month per gateway just for existing**, plus $0.045/GB processed. With gateway endpoints for S3 and interface endpoints for SQS/SSM, most service-to-AWS traffic bypasses NAT. The remaining NAT traffic is image pulls from ECR (use ECR VPC endpoint) and outbound to public internet (which should be rare from worker tasks). If a future engineer disables a VPC endpoint to debug something and forgets to re-enable it, the NAT bill explodes — flag this in the runbook for §10.4.

6. **SQS Fair Queue surcharge** (introduced July 2025): adds $0.10/M when MessageGroupId is set on a Standard queue message. Our architecture uses Standard queues without MessageGroupId, so this doesn't apply. Worth knowing about if FIFO-like ordering is later needed at any single stage — Fair Queue is cheaper than full FIFO ($0.50/M) and gets you per-group ordering on a Standard queue.

---

## Source verification trail

Prices captured from public AWS pricing pages and corroborating third-party guides, retrieved May 2026 (us-east-1 / N. Virginia):

| Service | Confirmed value | Sources reviewed |
|---|---|---|
| Fargate vCPU/hr | $0.04048 (x86), $0.03238 (ARM) | aws.amazon.com/fargate/pricing, vantage.sh, sedai.io, cloudtoggle.com, leanopstech.com |
| Fargate GB/hr | $0.004445 (x86), $0.003556 (ARM) | same |
| Fargate Spot discount | up to 70% off | same |
| S3 Standard | $0.023/GB-mo | aws docs, cloudchipr, cloudzero, go-cloud.io |
| S3 Standard-IA | $0.0125/GB-mo | same |
| S3 Glacier IR | $0.004/GB-mo | same |
| S3 PUT | $5/M | same |
| S3 GET | $0.40/M | same |
| S3 lifecycle <128KB exclusion | confirmed since Sept 2024 | cloudburn.io |
| SQS Standard | $0.40/M, first 1M free | aws docs, cloudburn, pump.co |
| SQS Fair Queue surcharge | +$0.10/M when MessageGroupId set | cloudburn |
| NAT egress | $0.045/GB processed + $0.045/hr | leanopstech, AWS networking docs |
| VPC gateway endpoint (S3, DynamoDB) | free | AWS networking docs |
| VPC interface endpoint | $0.01/hr/AZ + $0.01/GB | AWS networking docs |

For the v1 implementation, the cost calculator service (PoC #1, phase 7-8) should hit the live AWS Pricing API and refresh these values daily — don't trust this snapshot beyond June 2026 without re-verification.
