#!/usr/bin/env bash
set -euo pipefail
export THRONE_DATA_DIR="${THRONE_DATA_DIR:-.data}"
make proto
make migrate
make seed
(go run ./cmd/throne-ingest >"$THRONE_DATA_DIR/ingest.log" 2>&1 & echo $! >"$THRONE_DATA_DIR/ingest.pid")
sleep 1
go run ./cmd/throne-load --target http://localhost:8080 --devices 2 --rate-per-device 1 --duration 3s --payload-bytes 2048
kill "$(cat "$THRONE_DATA_DIR/ingest.pid")" || true
go run ./cmd/throne-processor --once
cat <<MSG
Demo data generated.
- API:        make run-api       then http://localhost:3000/healthz
- Cost API:   make run-cost-api  then http://localhost:9000/v1/pricing/fargate?region=us-east-1
- Calculator: cd web/cost && npm install && npm run dev
- Grafana:    docker compose up -d grafana prometheus tempo && open http://localhost:3001
MSG
