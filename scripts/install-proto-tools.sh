#!/usr/bin/env bash
set -euo pipefail
export PATH="${GOBIN:-$(go env GOPATH)/bin}:$PATH"

if ! command -v buf >/dev/null 2>&1; then
  echo "buf is not installed; install it from https://buf.build/docs/installation"
  exit 1
else
  buf --version
fi

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.3
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.3

for tool in protoc-gen-go protoc-gen-go-grpc protoc-gen-grpc-gateway protoc-gen-openapiv2; do
  command -v "$tool"
done
