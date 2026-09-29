#!/bin/sh

# Release the test lock created by acquire-lock.sh.
# Called from the Makefile EXIT trap: bash scripts/release-lock.sh $$

LOCK_FILE="${PROCHUB_LOCK_FILE:-_temp/dev-seed-test.lock}"
PID="${1:-}"

[ -f "$LOCK_FILE" ] || exit 0

# Only the owner may remove the lock file.
OWNER="$(head -1 "$LOCK_FILE" 2>/dev/null || true)"
if [ -z "$PID" ] || [ "$OWNER" = "$PID" ]; then
  rm -f "$LOCK_FILE"
fi
