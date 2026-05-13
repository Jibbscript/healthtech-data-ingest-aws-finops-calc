# ADR-0004: ECS/Fargate over EKS

## Context
The system needs container orchestration with low operational load.

## Decision
Use ECS/Fargate for PoC and early production.

## Consequences
The system pays a Fargate premium in exchange for no node management. Revisit EC2-backed capacity when traffic and team size justify it.
