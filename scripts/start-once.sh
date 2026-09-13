#!/usr/bin/env bash
# One-shot run without watching files.
set -euo pipefail
cd "$(dirname "$0")/.."
exec go run ./cmd/web
