# ADR-0002: Postgres over DynamoDB

## Context
Users, devices, captures, findings, and audit data require queryable metadata storage.

## Decision
Use RDS Postgres for metadata and findings.

## Consequences
The team keeps SQL joins and transactions, accepts RDS upgrade/backup operations, and can add read replicas before considering sharding.
