# ADR-0005: One queue per stage

## Context
Pipeline failures should be diagnosable by stage.

## Decision
Use one SQS queue and one DLQ per processing stage.

## Consequences
There are more queues to provision and monitor, but DLQ messages retain clear failure context.
