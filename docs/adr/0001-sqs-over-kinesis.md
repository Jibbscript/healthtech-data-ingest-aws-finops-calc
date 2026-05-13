# ADR-0001: SQS over Kinesis or MSK

## Context
The ingest service needs an async handoff to processing.

## Decision
Use SQS standard queues for v1.

## Consequences
Ordering and replay are not available by default, so consumers must be idempotent and failed captures must be handled through DLQ redrive. Revisit if ordered processing, replay, or >10M messages/day becomes necessary.
