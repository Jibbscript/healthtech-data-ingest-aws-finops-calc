# ADR-0008: Load generator as a first-class artifact

## Context
Scale and cost claims require repeatable evidence.

## Decision
Keep a configurable load generator in the repo and run a small version nightly.

## Consequences
The repo owns another executable, but architecture claims become testable instead of narrative-only.
