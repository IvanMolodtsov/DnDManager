#!/usr/bin/env bash
# Hot reload via Air. Rebuilds on .go / .html / .json / .sql / .css changes.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p tmp
echo "Hot reload: http://127.0.0.1:8080  (Ctrl+C to stop)"
exec go run github.com/air-verse/air@v1.61.7 \
  --build.cmd "go build -o ./tmp/web ./cmd/web" \
  --build.bin "./tmp/web"
