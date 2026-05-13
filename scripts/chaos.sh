#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-help}"; arg="${2:-}"
case "$cmd" in
  kill-processor|--kill)
    pkill -f throne-processor || true
    echo "processor stopped; queue age/depth should rise under load"
    ;;
  saturate-queue|--saturate)
    n="${arg:-1000}"
    for i in $(seq 1 "$n"); do
      mkdir -p .data/sqs/throne-ingest-jobs/pending
      printf '{"id":"chaos-%s","job":{"capture_id":"chaos-%s","device_id":"device_demo","user_id":"user_demo","s3_key":"missing","captured_at":"%s"}}\n' "$i" "$i" "$(date -u +%FT%TZ)" > ".data/sqs/throne-ingest-jobs/pending/chaos-$i.json"
    done
    echo "queued $n synthetic messages"
    ;;
  lag-inference|--lag)
    ms="${arg:-30000}"
    echo "set INFERENCE_LATENCY_MS=$ms before starting throne-inference to simulate lag"
    ;;
  kill-postgres)
    docker compose stop postgres || true
    ;;
  restore)
    docker compose up -d postgres localstack prometheus grafana tempo || true
    ;;
  *) echo "usage: $0 {kill-processor|saturate-queue N|lag-inference MS|kill-postgres|restore}" ;;
esac
