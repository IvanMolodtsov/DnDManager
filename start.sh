#!/usr/bin/env bash
# Convenience wrapper so `./start.sh` works from the repo root.
exec "$(dirname "$0")/scripts/start.sh" "$@"
