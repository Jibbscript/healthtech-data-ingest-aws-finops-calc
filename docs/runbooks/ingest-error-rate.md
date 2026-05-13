# Runbook: Ingest error rate > 1%

**Symptom:** `IngestErrorRateHigh` fires; device uploads are failing.

**Check:** Open `Throne Overview`, inspect status-code split, latest deploy, S3/SQS errors, and ALB target health.

**Mitigation:** Roll back the ingest task image if the spike follows deploy; otherwise scale ingest tasks to max and verify S3/SQS endpoint health.

**Root cause path:** Correlate failing trace IDs with logs, then classify as client auth, payload validation, S3 PutObject, SQS SendMessage, or infrastructure/network failure.
