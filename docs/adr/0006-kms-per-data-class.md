# ADR-0006: KMS per data class

## Context
PHI, standard, and telemetry data need different access boundaries.

## Decision
Create one KMS key per data class per environment.

## Consequences
IAM can grant decryption narrowly and PHI keys can rotate independently, at the cost of a small number of additional KMS keys.
