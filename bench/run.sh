#!/bin/sh
# websocketd Performance Benchmark Runner
#
# Usage: ./bench/run.sh <websocketd-binary> [options]
#
# Options:
#   --scenarios=LIST   Comma-separated scenario names (default: all)
#   --output=DIR       Output directory (default: bench/results)
#   --k6=PATH          Path to k6 binary (default: k6)
#
# Examples:
#   ./bench/run.sh ./websocketd
#   ./bench/run.sh ./websocketd --scenarios=echo_latency,echo_throughput
#   ./bench/run.sh /usr/local/bin/websocketd-0.4.1

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
K6="${K6:-k6}"
OUTPUT_DIR=""
SCENARIOS=""

# --- Argument parsing ---

WEBSOCKETD_BIN=""
for arg in "$@"; do
  case "$arg" in
    --scenarios=*) SCENARIOS="${arg#--scenarios=}" ;;
    --output=*)    OUTPUT_DIR="${arg#--output=}" ;;
    --k6=*)        K6="${arg#--k6=}" ;;
    --help|-h)
      sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      if [ -z "$WEBSOCKETD_BIN" ]; then
        WEBSOCKETD_BIN="$arg"
      else
        echo "Unknown argument: $arg" >&2
        exit 1
      fi
      ;;
  esac
done

if [ -z "$WEBSOCKETD_BIN" ]; then
  echo "Usage: $0 <websocketd-binary> [options]" >&2
  exit 1
fi

if [ ! -x "$WEBSOCKETD_BIN" ]; then
  echo "Error: $WEBSOCKETD_BIN is not executable" >&2
  exit 1
fi

command -v "$K6" >/dev/null 2>&1 || { echo "Error: k6 not found. Install: brew install k6" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "Error: python3 not found" >&2; exit 1; }

# --- Setup output directory ---

VERSION=$("$WEBSOCKETD_BIN" --version 2>&1 | head -1 || echo "unknown")
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
if [ -z "$OUTPUT_DIR" ]; then
  OUTPUT_DIR="$SCRIPT_DIR/results"
fi
mkdir -p "$OUTPUT_DIR"

# --- Write metadata ---

GIT_HASH=$(git -C "$(dirname "$WEBSOCKETD_BIN")" rev-parse --short HEAD 2>/dev/null || echo "unknown")
cat > "$OUTPUT_DIR/meta.json" <<METAEOF
{
  "version": "$(echo "$VERSION" | tr '"' "'")",
  "git_hash": "$GIT_HASH",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "os": "$(uname -s)",
  "arch": "$(uname -m)",
  "k6_version": "$($K6 version 2>&1 | head -1)"
}
METAEOF

echo "=== websocketd benchmark ==="
echo "Binary:  $WEBSOCKETD_BIN"
echo "Version: $VERSION"
echo "Output:  $OUTPUT_DIR"
echo ""

# --- Helper functions ---

find_free_port() {
  python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()"
}

wait_for_port() {
  local port=$1
  local timeout=${2:-10}
  local elapsed=0
  while ! python3 -c "import socket; s=socket.socket(); s.settimeout(0.2); s.connect(('127.0.0.1',$port)); s.close()" 2>/dev/null; do
    sleep 0.1
    elapsed=$((elapsed + 1))
    if [ "$elapsed" -ge "$((timeout * 10))" ]; then
      echo "Error: websocketd did not start on port $port within ${timeout}s" >&2
      return 1
    fi
  done
}

# Run a single scenario.
# Usage: run_scenario <name> <k6_script> [extra_ws_flags...] [-- k6_env_args...]
run_scenario() {
  local name="$1"
  local script="$2"
  shift 2

  # Split websocketd flags from k6 env args at "--"
  local ws_flags=""
  local k6_env=""
  local backend="$SCRIPT_DIR/backends/echo.sh"
  local past_separator=0

  for arg in "$@"; do
    if [ "$arg" = "--" ]; then
      past_separator=1
      continue
    fi
    if [ "$past_separator" -eq 0 ]; then
      case "$arg" in
        --backend=*) backend="${arg#--backend=}" ;;
        *) ws_flags="$ws_flags $arg" ;;
      esac
    else
      k6_env="$k6_env -e $arg"
    fi
  done

  local port=$(find_free_port)
  echo "--- $name (port $port) ---"

  # Start websocketd
  $WEBSOCKETD_BIN --port="$port" --address=127.0.0.1 $ws_flags "$backend" \
    >/dev/null 2>"$OUTPUT_DIR/${name}_ws_stderr.log" &
  local ws_pid=$!

  if ! wait_for_port "$port" 10; then
    kill "$ws_pid" 2>/dev/null; wait "$ws_pid" 2>/dev/null
    echo "SKIP: $name (server failed to start)"
    return 1
  fi

  # Start metrics collector
  "$SCRIPT_DIR/lib/collect-metrics.sh" "$ws_pid" "$OUTPUT_DIR/${name}_server.ndjson" &
  local collector_pid=$!

  # Run k6
  $K6 run \
    --summary-export="$OUTPUT_DIR/${name}_summary.json" \
    --out "json=$OUTPUT_DIR/${name}_k6.json" \
    -e "WS_PORT=$port" \
    $k6_env \
    --quiet \
    "$SCRIPT_DIR/scenarios/$script" 2>&1 | sed "s/^/  /"

  local k6_exit=$?

  # Stop websocketd and collector (suppress exit-on-signal noise)
  kill "$ws_pid" 2>/dev/null
  wait "$ws_pid" 2>/dev/null || true
  kill "$collector_pid" 2>/dev/null
  wait "$collector_pid" 2>/dev/null || true

  if [ "$k6_exit" -ne 0 ]; then
    echo "  WARN: k6 exited with code $k6_exit"
  fi
  echo ""
}

# --- Define default scenarios ---

ALL_SCENARIOS="echo_latency echo_throughput connection_storm_10 connection_storm_100 connection_storm_500 connection_churn sustained_load binary_1k binary_10k binary_64k backpressure"

if [ -n "$SCENARIOS" ]; then
  RUN_SCENARIOS=$(echo "$SCENARIOS" | tr ',' ' ')
else
  RUN_SCENARIOS="$ALL_SCENARIOS"
fi

# --- Run scenarios ---

for scenario in $RUN_SCENARIOS; do
  case "$scenario" in
    echo_latency)
      run_scenario "echo_latency" "echo_latency.js"
      ;;
    echo_throughput)
      run_scenario "echo_throughput" "echo_throughput.js"
      ;;
    connection_storm_10)
      run_scenario "connection_storm_10" "connection_storm.js" -- "STORM_VUS=10"
      ;;
    connection_storm_100)
      run_scenario "connection_storm_100" "connection_storm.js" -- "STORM_VUS=100"
      ;;
    connection_storm_500)
      run_scenario "connection_storm_500" "connection_storm.js" -- "STORM_VUS=500"
      ;;
    connection_churn)
      run_scenario "connection_churn" "connection_churn.js"
      ;;
    sustained_load)
      run_scenario "sustained_load" "sustained_load.js"
      ;;
    binary_1k)
      run_scenario "binary_1k" "binary_throughput.js" --binary "--backend=$SCRIPT_DIR/backends/binary-echo.sh" -- "PAYLOAD_SIZE=1024"
      ;;
    binary_10k)
      run_scenario "binary_10k" "binary_throughput.js" --binary "--backend=$SCRIPT_DIR/backends/binary-echo.sh" -- "PAYLOAD_SIZE=10240"
      ;;
    binary_64k)
      run_scenario "binary_64k" "binary_throughput.js" --binary "--backend=$SCRIPT_DIR/backends/binary-echo.sh" -- "PAYLOAD_SIZE=65536"
      ;;
    backpressure)
      run_scenario "backpressure" "backpressure.js" "--backend=$SCRIPT_DIR/backends/slow-consumer.sh"
      ;;
    *)
      echo "Unknown scenario: $scenario" >&2
      ;;
  esac
done

# --- Generate reports ---

echo "=== Generating reports ==="
python3 "$SCRIPT_DIR/lib/to-benchmark-action.py" "$OUTPUT_DIR"
python3 "$SCRIPT_DIR/lib/generate-report.py" "$OUTPUT_DIR"

echo ""
echo "Done! Results in $OUTPUT_DIR/"
echo "  report.html          — open in browser"
echo "  benchmark-data.json  — for CI regression detection"
