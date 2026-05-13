# ADR-0007: S3 lifecycle policies in Terraform

## Context
Storage tiering is the largest cost lever.

## Decision
Define lifecycle transitions in Terraform, not application code.

## Consequences
Cost-affecting retention changes require reviewed infrastructure changes and are visible to auditors.
