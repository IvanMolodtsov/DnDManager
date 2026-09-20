#!/bin/sh
# Copy repo-root assets into api/ so @vercel/go includeFiles (cwd = api/) can pack them.
# Do not use ../ globs — Vercel rejects those file descriptors.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
for d in web locales migrations; do
	rm -rf "api/$d"
	cp -R "$d" "api/$d"
done
