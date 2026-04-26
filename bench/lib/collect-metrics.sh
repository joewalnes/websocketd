#!/bin/sh
# Collect server-side metrics (RSS, FD count) by sampling ps.
# Usage: collect-metrics.sh <pid> <output.ndjson>
# Writes one JSON object per line every 500ms until the process exits.

PID="$1"
OUTPUT="$2"

if [ -z "$PID" ] || [ -z "$OUTPUT" ]; then
  echo "Usage: collect-metrics.sh <pid> <output.ndjson>" >&2
  exit 1
fi

> "$OUTPUT"

while kill -0 "$PID" 2>/dev/null; do
  TIMESTAMP=$(python3 -c "import time; print(int(time.time()*1000))")
  RSS=$(ps -o rss= -p "$PID" 2>/dev/null | tr -d ' ')
  if [ -d "/proc/$PID/fd" ]; then
    FDS=$(ls /proc/$PID/fd 2>/dev/null | wc -l | tr -d ' ')
  else
    FDS=$(lsof -p "$PID" 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
  fi
  printf '{"ts":%s,"rss_kb":%s,"fds":%s}\n' "${TIMESTAMP}" "${RSS:-0}" "${FDS:-0}" >> "$OUTPUT"
  sleep 0.5
done
