# ADR-0003: gRPC primary with REST gateway

## Context
Devices prefer gRPC while mobile/web clients need REST.

## Decision
Define the canonical contract in protobuf and generate REST through grpc-gateway.

## Consequences
The repo has a generation step, but avoids hand-maintaining two API contracts.
