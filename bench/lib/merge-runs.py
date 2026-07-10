#!/usr/bin/env python3
"""Merge N repeated k6 runs of the same scenario into one median result.

Reduces run-to-run noise on shared CI hardware: a single k6 run's numbers
can vary 20%+ between otherwise-identical commits, which previously
produced false-positive regression alerts. Instead, each scenario now runs
N times (run.sh) and this script:

  - Takes the per-metric median across all N *_summary.run{i}.json files
    (recursively, matching by JSON path - robust to new metrics without
    code changes here) and writes it to *_summary.json, which is what
    to-benchmark-action.py and generate-report.py already read.
  - Picks the run whose peak RSS is closest to the group's median and
    copies its *_server.run{i}.ndjson to *_server.ndjson, so the HTML
    report's memory/FD chart still shows one real, representative time
    series instead of a synthetic one.
"""

import json
import os
import statistics
import sys


def median_merge(values):
    """Recursively merge parsed JSON values of identical shape, taking the
    median of numeric leaves. Non-numeric or shape-mismatched leaves fall
    back to the first run's value."""
    first = values[0]

    if isinstance(first, bool):
        return first  # bool is an int subclass in Python; keep as-is

    if isinstance(first, (int, float)):
        nums = [v for v in values if isinstance(v, (int, float)) and not isinstance(v, bool)]
        if len(nums) == len(values):
            return statistics.median(nums)
        return first

    if isinstance(first, dict):
        merged = {}
        for key in first:
            key_values = [v[key] if isinstance(v, dict) and key in v else first[key] for v in values]
            merged[key] = median_merge(key_values)
        return merged

    if isinstance(first, list):
        # Lists (e.g. threshold failure details) aren't aligned in any
        # meaningful way across separate runs; keep the first run's list.
        return first

    return first


def peak_rss(ndjson_path):
    peak = 0
    if not os.path.exists(ndjson_path):
        return peak
    with open(ndjson_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                peak = max(peak, json.loads(line).get("rss_kb", 0))
            except json.JSONDecodeError:
                continue
    return peak


def pick_median_run(ndjson_paths):
    """Return the path whose peak RSS is the median of the group."""
    existing = [p for p in ndjson_paths if os.path.exists(p)]
    if not existing:
        return None
    by_peak = sorted((peak_rss(p), p) for p in existing)
    return by_peak[len(by_peak) // 2][1]


def main():
    if len(sys.argv) < 4:
        print("Usage: merge-runs.py <results-dir> <scenario-name> <num-runs>", file=sys.stderr)
        sys.exit(1)

    results_dir, name, num_runs = sys.argv[1], sys.argv[2], int(sys.argv[3])

    summary_paths = [
        os.path.join(results_dir, f"{name}_summary.run{i}.json")
        for i in range(1, num_runs + 1)
    ]
    summary_paths = [p for p in summary_paths if os.path.exists(p)]
    if not summary_paths:
        print(f"  No summary files found for {name}, skipping merge", file=sys.stderr)
        return

    summaries = [json.load(open(p)) for p in summary_paths]
    merged = median_merge(summaries)
    with open(os.path.join(results_dir, f"{name}_summary.json"), "w") as f:
        json.dump(merged, f, indent=2)

    ndjson_paths = [
        os.path.join(results_dir, f"{name}_server.run{i}.ndjson")
        for i in range(1, num_runs + 1)
    ]
    chosen = pick_median_run(ndjson_paths)
    if chosen:
        with open(chosen) as src, open(os.path.join(results_dir, f"{name}_server.ndjson"), "w") as dst:
            dst.write(src.read())

    print(f"  Merged {len(summary_paths)} runs for {name} (median)")


if __name__ == "__main__":
    main()
