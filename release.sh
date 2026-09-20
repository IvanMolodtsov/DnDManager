#!/usr/bin/env bash
# Convenience wrapper so `./release.sh` works from the repo root.
exec "$(dirname "$0")/scripts/release.sh" "$@"
