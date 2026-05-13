# Runbook: Queue age > 60s

**Symptom:** `QueueAgeHigh` fires; captures are accepted but findings are delayed.

**Check:** Open `Throne Pipeline Health`; compare queue age, processor throughput, inference p99, and processor task count.

**Mitigation:** Scale processor desired count up, confirm inference latency is below 30s, and pause load tests if this is a synthetic run.

**Root cause path:** If throughput is zero, inspect processor logs and task health. If throughput is nonzero but low, inspect inference saturation and database writes.
