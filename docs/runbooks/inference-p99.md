# Runbook: Inference p99 > 30s

**Symptom:** `InferenceP99High` fires; end-to-end freshness is at risk.

**Check:** Inspect inference p99 by model version, inflight count, CPU/memory, and processor queue age.

**Mitigation:** Scale inference tasks, roll back a new model version if latency regressed, or temporarily lower processor concurrency to prevent overload.

**Root cause path:** Compare model version, image digest, payload size distribution, and cold-start frequency before changing autoscaling thresholds.
