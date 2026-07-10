#!/bin/sh
# websocketd Performance Benchmark Runner
#
# Usage: ./bench/run.sh <websocketd-binary> [options]
#
# Options:
#   --scenarios=LIST   Comma-separated scenario names (default: all)
#   --output=DIR       Output directory (default: bench/results)
#   --k6=PATH          Path to k6 binary (default: k6)
#   --runs=N           Repeat each scenario N times and report the median
#                      (default: 3). Shared CI hardware is noisy enough that
#                      a single k6 run can vary 20%+ between otherwise-
#                      identical commits; the median of a few runs is far
#                      more stable.
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
RUNS_PER_SCENARIO=3

# --- Argument parsing ---

WEBSOCKETD_BIN=""
for arg in "$@"; do
  case "$arg" in
    --scenarios=*) SCENARIOS="${arg#--scenarios=}" ;;
    --output=*)    OUTPUT_DIR="${arg#--output=}" ;;
    --k6=*)        K6="${arg#--k6=}" ;;
    --runs=*)      RUNS_PER_SCENARIO="${arg#--runs=}" ;;
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

# Collect machine specs
OS_NAME=$(uname -s)
OS_VERSION=$(uname -r)
ARCH=$(uname -m)
if [ "$OS_NAME" = "Darwin" ]; then
  CPU_MODEL=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "unknown")
  CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null || echo "?")
  RAM_BYTES=$(sysctl -n hw.memsize 2>/dev/null || echo "0")
  RAM_GB=$(python3 -c "print(round($RAM_BYTES / (1024**3), 1))" 2>/dev/null || echo "?")
  OS_PRETTY=$(sw_vers -productName 2>/dev/null || echo "macOS")
  OS_PRETTY="$OS_PRETTY $(sw_vers -productVersion 2>/dev/null || echo "$OS_VERSION")"
elif [ "$OS_NAME" = "Linux" ]; then
  CPU_MODEL=$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | xargs || echo "unknown")
  CPU_CORES=$(nproc 2>/dev/null || echo "?")
  RAM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo "0")
  RAM_GB=$(python3 -c "print(round($RAM_KB / (1024**2), 1))" 2>/dev/null || echo "?")
  OS_PRETTY=$(cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'"' -f2 || echo "Linux $OS_VERSION")
else
  CPU_MODEL="unknown"
  CPU_CORES="?"
  RAM_GB="?"
  OS_PRETTY="$OS_NAME $OS_VERSION"
fi

cat > "$OUTPUT_DIR/meta.json" <<METAEOF
{
  "version": "$(echo "$VERSION" | tr '"' "'")",
  "git_hash": "$GIT_HASH",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "os": "$OS_NAME",
  "os_pretty": "$OS_PRETTY",
  "arch": "$ARCH",
  "cpu": "$CPU_MODEL",
  "cpu_cores": "$CPU_CORES",
  "ram_gb": "$RAM_GB",
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

# Run one iteration of a scenario, writing to run-suffixed output files.
# Usage: run_scenario_once <run_index> <name> <script> <backend> <ws_flags> <k6_env>
run_scenario_once() {
  local run_index="$1"
  local name="$2"
  local script="$3"
  local backend="$4"
  local ws_flags="$5"
  local k6_env="$6"

  local port=$(find_free_port)
  echo "--- $name run $run_index/$RUNS_PER_SCENARIO (port $port) ---"

  # Start websocketd
  $WEBSOCKETD_BIN --port="$port" --address=127.0.0.1 $ws_flags "$backend" \
    >/dev/null 2>"$OUTPUT_DIR/${name}_ws_stderr.run${run_index}.log" &
  local ws_pid=$!

  if ! wait_for_port "$port" 10; then
    kill "$ws_pid" 2>/dev/null; wait "$ws_pid" 2>/dev/null
    echo "SKIP: $name run $run_index (server failed to start)"
    return 1
  fi

  # Start metrics collector
  "$SCRIPT_DIR/lib/collect-metrics.sh" "$ws_pid" "$OUTPUT_DIR/${name}_server.run${run_index}.ndjson" &
  local collector_pid=$!

  # Run k6. Output goes to a log first: piping k6 straight into sed would
  # make $? report sed's status, silently discarding k6 failures.
  local k6_exit=0
  $K6 run \
    --summary-export="$OUTPUT_DIR/${name}_summary.run${run_index}.json" \
    --out "json=$OUTPUT_DIR/${name}_k6.run${run_index}.json" \
    -e "WS_PORT=$port" \
    $k6_env \
    --quiet \
    "$SCRIPT_DIR/scenarios/$script" >"$OUTPUT_DIR/${name}_k6.run${run_index}.log" 2>&1 || k6_exit=$?
  sed "s/^/  /" "$OUTPUT_DIR/${name}_k6.run${run_index}.log"

  # Stop websocketd and collector (suppress exit-on-signal noise)
  kill "$ws_pid" 2>/dev/null
  wait "$ws_pid" 2>/dev/null || true
  kill "$collector_pid" 2>/dev/null
  wait "$collector_pid" 2>/dev/null || true

  if [ "$k6_exit" -eq 99 ]; then
    # k6 exits 99 when thresholds are crossed — advisory on shared hardware.
    echo "  WARN: $name run $run_index crossed k6 thresholds"
  elif [ "$k6_exit" -ne 0 ]; then
    echo "  ERROR: $name run $run_index exited with code $k6_exit"
    return 1
  fi
  return 0
}

# Run a scenario RUNS_PER_SCENARIO times and merge the results into a single
# median (see lib/merge-runs.py). A single k6 run on shared CI hardware is
# noisy enough to produce false-positive regression alerts; repeating and
# taking the median is far more stable without needing dedicated hardware.
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

  local run_index=1
  local any_failed=0
  while [ "$run_index" -le "$RUNS_PER_SCENARIO" ]; do
    if ! run_scenario_once "$run_index" "$name" "$script" "$backend" "$ws_flags" "$k6_env"; then
      any_failed=1
    fi
    run_index=$((run_index + 1))
  done
  echo ""

  if [ "$any_failed" -eq 1 ]; then
    echo "  FAILED: $name (at least one of $RUNS_PER_SCENARIO runs failed)"
    FAILED_SCENARIOS="$FAILED_SCENARIOS $name"
    return 0 # returning non-zero would abort the whole run under set -e
  fi

  python3 "$SCRIPT_DIR/lib/merge-runs.py" "$OUTPUT_DIR" "$name" "$RUNS_PER_SCENARIO"
}

# --- Define default scenarios ---

FAILED_SCENARIOS=""
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

if [ -n "$FAILED_SCENARIOS" ]; then
  echo ""
  echo "FAILED scenarios:$FAILED_SCENARIOS" >&2
  exit 1
fi
