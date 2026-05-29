#!/usr/bin/env bash
# Sync the canonical skills/ bundle into the npm wrapper package (npm/skills/).
#
# skills/ at the repo root is the single source of truth. The npm package ships
# its own copy so `npx @rahadiangg/bbx-skills` works without the Go binary. This
# script keeps the copy in lockstep.
#
#   scripts/sync-npm-skills.sh           # copy skills/ -> npm/skills/
#   scripts/sync-npm-skills.sh --check   # verify they're identical (CI gate)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/skills"
DST="$ROOT/npm/skills"

if [[ "${1:-}" == "--check" ]]; then
  if ! diff -r "$SRC" "$DST" >/dev/null 2>&1; then
    echo "npm/skills is out of sync with skills/." >&2
    echo "Run: make sync-npm" >&2
    diff -r "$SRC" "$DST" || true
    exit 1
  fi
  echo "npm/skills is in sync with skills/."
  exit 0
fi

rm -rf "$DST"
mkdir -p "$DST"
cp -R "$SRC"/. "$DST"/
echo "Synced skills/ -> npm/skills/"
