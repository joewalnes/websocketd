#!/usr/bin/env python3
"""Convert k6 summary JSON files to benchmark-action format.

Reads *_summary.json files from the results directory and produces
benchmark-data.json compatible with benchmark-action/github-action-benchmark
using the 'customSmallerIsBetter' tool type.

Throughput metrics are inverted (reported as µs/msg) so all metrics
are "smaller is better".
"""

import json
import glob
import os
import sys


def extract_trend(summary, metric_name):
    """Extract p50/p95/p99/avg from a k6 Trend metric.

    k6 --summary-export puts values directly on the metric object:
      {"avg": 0.054, "min": 0, "med": 0, "max": 4, "p(90)": 0, "p(95)": 1}
    """
    metrics = summary.get("metrics", {})
    m = metrics.get(metric_name, {})
    if not m:
        return {}
    # k6 puts values either directly or under "values"
    v = m.get("values", m)
    return {
        "avg": v.get("avg", 0),
        "p50": v.get("med", 0),
        "p95": v.get("p(95)", 0),
        "p99": v.get("p(99)", 0),
    }


def extract_counter(summary, metric_name):
    """Extract the count value from a k6 Counter metric."""
    metrics = summary.get("metrics", {})
    m = metrics.get(metric_name, {})
    if not m:
        return 0
    v = m.get("values", m)
    return v.get("count", 0)


def extract_rate_per_sec(summary, metric_name):
    """Extract per-second rate from a k6 Counter metric."""
    metrics = summary.get("metrics", {})
    m = metrics.get(metric_name, {})
    if not m:
        return 0
    v = m.get("values", m)
    return v.get("rate", 0)


def peak_rss(results_dir, scenario_name):
    """Read peak RSS from server metrics NDJSON."""
    path = os.path.join(results_dir, f"{scenario_name}_server.ndjson")
    peak = 0
    if os.path.exists(path):
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                    peak = max(peak, obj.get("rss_kb", 0))
                except json.JSONDecodeError:
                    continue
    return peak


def process_scenario(results_dir, name, benchmarks):
    """Process a single scenario's summary JSON."""
    path = os.path.join(results_dir, f"{name}_summary.json")
    if not os.path.exists(path):
        return

    with open(path) as f:
        summary = json.load(f)

    # Latency scenarios
    if name == "echo_latency":
        trend = extract_trend(summary, "ws_rtt_ms")
        for stat in ["p50", "p95", "p99", "avg"]:
            if stat in trend:
                benchmarks.append({
                    "name": f"echo_latency_{stat}",
                    "unit": "ms",
                    "value": round(trend[stat], 3),
                })

    elif name == "echo_throughput":
        recv = extract_counter(summary, "ws_msgs_recv")
        # k6 iteration_duration gives total time
        dur = summary.get("metrics", {}).get("iteration_duration", {})
        dur_v = dur.get("values", dur)
        dur_ms = dur_v.get("avg", 12000)
        if recv > 0:
            msgs_per_sec = recv / (dur_ms / 1000)
            # Invert: µs per message (smaller is better)
            benchmarks.append({
                "name": "echo_throughput_us_per_msg",
                "unit": "µs/msg",
                "value": round(1_000_000 / msgs_per_sec, 3) if msgs_per_sec > 0 else 999999,
            })
            benchmarks.append({
                "name": "echo_throughput_msgs_sec",
                "unit": "msgs/sec (info only)",
                "value": round(msgs_per_sec, 0),
            })

    elif name.startswith("connection_storm_"):
        vus = name.split("_")[-1]
        trend = extract_trend(summary, "ws_cycle_time_ms")
        for stat in ["p95", "avg"]:
            if stat in trend:
                benchmarks.append({
                    "name": f"connection_storm_{vus}_{stat}",
                    "unit": "ms",
                    "value": round(trend[stat], 3),
                })

    elif name == "connection_churn":
        trend = extract_trend(summary, "ws_churn_cycle_ms")
        if "avg" in trend:
            benchmarks.append({
                "name": "connection_churn_avg_ms",
                "unit": "ms",
                "value": round(trend["avg"], 3),
            })
            if trend["avg"] > 0:
                benchmarks.append({
                    "name": "connection_churn_conns_sec",
                    "unit": "conn/sec (info only)",
                    "value": round(1000 / trend["avg"], 1),
                })

    elif name == "sustained_load":
        trend = extract_trend(summary, "ws_sustained_rtt_ms")
        for stat in ["p50", "p95", "p99"]:
            if stat in trend:
                benchmarks.append({
                    "name": f"sustained_load_rtt_{stat}",
                    "unit": "ms",
                    "value": round(trend[stat], 3),
                })
        recv = extract_counter(summary, "ws_sustained_msgs")
        if recv > 0:
            benchmarks.append({
                "name": "sustained_load_total_msgs",
                "unit": "msgs (info only)",
                "value": recv,
            })

    elif name.startswith("binary_"):
        size = name.split("_")[1]
        recv = extract_counter(summary, "ws_binary_bytes_recv")
        dur = summary.get("metrics", {}).get("iteration_duration", {})
        dur_ms = dur.get("values", {}).get("avg", 1000)
        if recv > 0 and dur_ms > 0:
            mbps = (recv / (1024 * 1024)) / (dur_ms / 1000)
            benchmarks.append({
                "name": f"binary_{size}_MB_sec",
                "unit": "MB/s (info only)",
                "value": round(mbps, 2),
            })

    elif name == "backpressure":
        recv = extract_counter(summary, "ws_bp_msgs_recv")
        sent = extract_counter(summary, "ws_bp_msgs_sent")
        benchmarks.append({
            "name": "backpressure_msgs_echoed",
            "unit": "msgs (info only)",
            "value": recv,
        })
        if sent > 0:
            benchmarks.append({
                "name": "backpressure_delivery_ratio",
                "unit": "ratio (info only)",
                "value": round(recv / sent, 4),
            })

    # Peak RSS for every scenario
    rss = peak_rss(results_dir, name)
    if rss > 0:
        benchmarks.append({
            "name": f"{name}_peak_rss_kb",
            "unit": "KB",
            "value": rss,
        })


def main():
    if len(sys.argv) < 2:
        print("Usage: to-benchmark-action.py <results-dir>", file=sys.stderr)
        sys.exit(1)

    results_dir = sys.argv[1]
    benchmarks = []

    # Find all summary files
    for path in sorted(glob.glob(os.path.join(results_dir, "*_summary.json"))):
        name = os.path.basename(path).replace("_summary.json", "")
        process_scenario(results_dir, name, benchmarks)

    output_path = os.path.join(results_dir, "benchmark-data.json")
    with open(output_path, "w") as f:
        json.dump(benchmarks, f, indent=2)

    print(f"  Wrote {len(benchmarks)} metrics to {output_path}")


if __name__ == "__main__":
    main()
