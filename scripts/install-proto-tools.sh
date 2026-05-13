#!/usr/bin/env bash
set -euo pipefail
if ! command -v buf >/dev/null 2>&1; then
  echo "buf is not installed; install from https://buf.build/docs/installation or use make proto for local deterministic stubs"
else
  buf --version
fi
for tool in protoc-gen-go protoc-gen-go-grpc protoc-gen-grpc-gateway protoc-gen-openapiv2; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool missing; install it if you need real generated gRPC/gateway code"
  fi
done
