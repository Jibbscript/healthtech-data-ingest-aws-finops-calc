# Runbook: DLQ depth > 0

**Symptom:** `DLQDepthNonZero` fires; at least one capture failed all retries.

**Check:** Inspect DLQ message bodies and related capture IDs. Check whether errors are permanent payload failures or transient dependency failures.

**Mitigation:** For transient failures, redrive after the dependency is healthy. For permanent failures, mark capture failed and preserve the message for incident review.

**Root cause path:** Trace capture ID through ingest, processor, inference, and Postgres writes; add a regression test before changing retry/redrive behavior.
