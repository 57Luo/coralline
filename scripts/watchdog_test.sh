#!/usr/bin/env bash
# Watchdog verification (go-renderer-core task 4.1).
#
# Requirement "Hard process deadline (watchdog)": started with a stdin pipe whose
# write end is held open and never closed, the renderer must exit on its own
# within 5 seconds with exit code 1 and empty stdout — it must never become a
# zombie waiting on an EOF that never arrives.
#
# Usage: scripts/watchdog_test.sh [path-to-coralline.exe]
# Exits 0 on pass, 1 on failure.
set -u

EXE="${1:-./coralline.exe}"
if [ ! -x "$EXE" ]; then
  echo "FAIL: executable not found: $EXE" >&2
  exit 1
fi

out="$(mktemp)"
err="$(mktemp)"
trap 'rm -f "$out" "$err"' EXIT

# Process substitution supplies a stdin fd connected to `sleep`, whose write end
# stays open and delivers no data and no EOF for 20s. The renderer's bounded
# stdin read blocks on it, so only the in-process watchdog can end the process.
# The parent shell returns as soon as the renderer exits (it does not wait for
# the backgrounded sleep), so a passing run completes in ~5s, not 20s.
start=$SECONDS
"$EXE" < <(sleep 20) > "$out" 2> "$err"
code=$?
elapsed=$(( SECONDS - start ))

fail=0
if [ "$code" -ne 1 ]; then
  echo "FAIL: exit code = $code (want 1)" >&2
  fail=1
fi
# SECONDS has 1s granularity; a correct 5s deadline lands in [4,7].
if [ "$elapsed" -lt 4 ] || [ "$elapsed" -gt 7 ]; then
  echo "FAIL: elapsed ${elapsed}s (want ~5, in [4,7])" >&2
  fail=1
fi
if [ -s "$out" ]; then
  echo "FAIL: stdout is not empty:" >&2
  cat "$out" >&2
  fail=1
fi

if [ "$fail" -eq 0 ]; then
  echo "PASS: watchdog exited in ${elapsed}s with code $code and empty stdout"
fi
exit "$fail"
