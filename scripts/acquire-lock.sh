#!/bin/sh

# Acquire an exclusive lock before running the automated tests.
# Sourced by the Makefile:
#   . scripts/acquire-lock.sh "dev-seed-test"
#
# Lock file lives in _temp (<name>.lock) and its first line holds the owner PID:
#   - owner still alive  -> print its PID and wait until it finishes;
#   - owner already gone -> treat the lock as stale, ignore it and take over.

LOCK_NAME="${1:-test}"
LOCK_ROOT="${LOCK_ROOT:-_temp}"
mkdir -p "$LOCK_ROOT"
LOCK_FILE="${LOCK_FILE:-$LOCK_ROOT/${LOCK_NAME}.lock}"
# Store an absolute path so the EXIT trap can release the lock from any cwd.
LOCK_FILE="$(cd "$(dirname "$LOCK_FILE")" && pwd)/$(basename "$LOCK_FILE")"
export PROCHUB_LOCK_FILE="$LOCK_FILE"

LAST_PID=""
while :; do
  # Create the lock file atomically; noclobber refuses to overwrite a live lock.
  if (set -C; printf '%s\n' "$$" > "$LOCK_FILE") 2>/dev/null; then
    echo "[lock] 已获取测试锁（PID=$$）"
    break
  fi

  LOCK_PID="$(head -1 "$LOCK_FILE" 2>/dev/null || true)"
  if [ -z "$LOCK_PID" ]; then
    # The owner may not have written its PID yet, give it a moment.
    sleep 1
    LOCK_PID="$(head -1 "$LOCK_FILE" 2>/dev/null || true)"
  fi

  if [ -z "$LOCK_PID" ] || ! kill -0 "$LOCK_PID" 2>/dev/null; then
    echo "[lock] 检测到残留锁（持有进程 PID ${LOCK_PID:-未知} 已退出），忽略锁并接管"
    rm -f "$LOCK_FILE"
    continue
  fi

  if [ "$LOCK_PID" != "$LAST_PID" ]; then
    echo "[lock] 其他进程正在进行测试（PID: ${LOCK_PID}），等待其结束后自动继续..."
    LAST_PID="$LOCK_PID"
  fi
  sleep 2
done
