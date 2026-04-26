# websocketd Performance Benchmarks

Automated performance benchmarking for websocketd using [k6](https://k6.io/).

## Quick Start

```bash
# Install k6 (macOS)
brew install k6

# Build and benchmark
go build -o websocketd .
./bench/run.sh ./websocketd
open bench/results/report.html
```

## Benchmarking Any Version

The benchmark tool works with any websocketd binary — use it to compare versions:

```bash
# Benchmark current build
./bench/run.sh ./websocketd --output=bench/results/v0.5.0

# Benchmark an older release
./bench/run.sh /usr/local/bin/websocketd-0.4.1 --output=bench/results/v0.4.1
```

## Scenarios

| Scenario | What It Measures |
|----------|-----------------|
| `echo_latency` | Round-trip time: 1 connection, 1000 sequential send/recv |
| `echo_throughput` | Max msgs/sec: 1 connection, fire-hose for 10s |
| `connection_storm` | Concurrent connect overhead: 10/100/500 VUs |
| `connection_churn` | Process lifecycle: 200 serial connect/send/close cycles |
| `sustained_load` | Steady state: 50 connections, continuous traffic for 30s |
| `binary_*` | Binary mode throughput: 1KB, 10KB, 64KB payloads |
| `backpressure` | Slow consumer: fast sender vs delayed echo backend |

## Metrics

### Client-Side (k6)
- **Latency**: p50, p95, p99, avg, max round-trip time (ms)
- **Throughput**: messages/sec, connections/sec
- **Binary**: MB/sec per payload size
- **Backpressure**: delivery ratio (received/sent)

### Server-Side (ps sampling)
- **RSS**: resident set size (KB) sampled every 500ms
- **FDs**: open file descriptors sampled every 500ms

## Output

```
bench/results/
  meta.json                    # Version, git hash, timestamp, OS
  *_summary.json               # k6 summary per scenario
  *_k6.json                    # k6 detailed output per scenario
  *_server.ndjson              # Server metrics per scenario
  report.html                  # Visual report (open in browser)
  benchmark-data.json          # CI regression detection format
```

## Running a Subset

```bash
./bench/run.sh ./websocketd --scenarios=echo_latency,echo_throughput
```

## CI Integration

The `bench.yml` GitHub Actions workflow:
- Runs all scenarios on every push to master and on PRs
- Uploads `report.html` as a workflow artifact
- Tracks metrics over time on [the dashboard](https://joewalnes.github.io/websocketd/dev/bench/)
- Posts comparison comments on PRs
- Blocks merges on >15% regression

## Design Decisions

- **`cat` as backend**: Measures websocketd overhead only, not backend processing time
- **k6**: Industry-standard load testing tool with native WebSocket support
- **Fresh process per scenario**: Each scenario gets a clean websocketd instance
- **"Smaller is better" normalization**: Throughput reported as µs/msg for unified CI thresholds
