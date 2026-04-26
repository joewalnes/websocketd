#!/usr/bin/env python3
"""Generate a self-contained HTML benchmark report from k6 results.

Reads *_summary.json and *_server.ndjson files from the results directory
and produces report.html with embedded Chart.js visualizations.
"""

import json
import glob
import os
import sys
from datetime import datetime


def load_summary(path):
    """Load a k6 summary JSON file."""
    with open(path) as f:
        return json.load(f)


def load_server_metrics(path):
    """Load server metrics NDJSON file."""
    metrics = []
    if not os.path.exists(path):
        return metrics
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                metrics.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return metrics


def extract_trend_values(summary, metric_name):
    """Extract all stat values from a k6 Trend metric."""
    metrics = summary.get("metrics", {})
    m = metrics.get(metric_name, {})
    if not m:
        return {}
    return m.get("values", m)


def extract_counter_value(summary, metric_name):
    """Extract count from a k6 Counter metric."""
    metrics = summary.get("metrics", {})
    m = metrics.get(metric_name, {})
    if not m:
        return 0
    v = m.get("values", m)
    return v.get("count", 0)


def build_report_data(results_dir):
    """Build the data structure for the HTML report."""
    data = {
        "meta": {},
        "scenarios": {},
        "server_metrics": {},
    }

    # Load metadata
    meta_path = os.path.join(results_dir, "meta.json")
    if os.path.exists(meta_path):
        with open(meta_path) as f:
            data["meta"] = json.load(f)

    # Process each scenario
    for path in sorted(glob.glob(os.path.join(results_dir, "*_summary.json"))):
        name = os.path.basename(path).replace("_summary.json", "")
        summary = load_summary(path)
        scenario = {"name": name}

        if name == "echo_latency":
            v = extract_trend_values(summary, "ws_rtt_ms")
            scenario["type"] = "latency"
            scenario["p50"] = round(v.get("med", 0), 3)
            scenario["p95"] = round(v.get("p(95)", 0), 3)
            scenario["p99"] = round(v.get("p(99)", 0), 3)
            scenario["avg"] = round(v.get("avg", 0), 3)
            scenario["min"] = round(v.get("min", 0), 3)
            scenario["max"] = round(v.get("max", 0), 3)

        elif name == "echo_throughput":
            recv = extract_counter_value(summary, "ws_msgs_recv")
            dur = summary.get("metrics", {}).get("iteration_duration", {})
            dur_v = dur.get("values", dur)
            dur_ms = dur_v.get("avg", 12000)
            scenario["type"] = "throughput"
            scenario["msgs_recv"] = recv
            scenario["duration_s"] = round(dur_ms / 1000, 1)
            scenario["msgs_per_sec"] = round(recv / (dur_ms / 1000), 0) if dur_ms > 0 else 0

        elif name.startswith("connection_storm_"):
            v = extract_trend_values(summary, "ws_cycle_time_ms")
            scenario["type"] = "storm"
            scenario["vus"] = int(name.split("_")[-1])
            scenario["p95"] = round(v.get("p(95)", 0), 3)
            scenario["avg"] = round(v.get("avg", 0), 3)
            scenario["max"] = round(v.get("max", 0), 3)

        elif name == "connection_churn":
            v = extract_trend_values(summary, "ws_churn_cycle_ms")
            scenario["type"] = "churn"
            scenario["avg"] = round(v.get("avg", 0), 3)
            scenario["p95"] = round(v.get("p(95)", 0), 3)
            scenario["conns_per_sec"] = round(1000 / v["avg"], 1) if v.get("avg", 0) > 0 else 0

        elif name == "sustained_load":
            v = extract_trend_values(summary, "ws_sustained_rtt_ms")
            recv = extract_counter_value(summary, "ws_sustained_msgs")
            scenario["type"] = "sustained"
            scenario["p50"] = round(v.get("med", 0), 3)
            scenario["p95"] = round(v.get("p(95)", 0), 3)
            scenario["p99"] = round(v.get("p(99)", 0), 3)
            scenario["total_msgs"] = recv

        elif name.startswith("binary_"):
            recv = extract_counter_value(summary, "ws_binary_bytes_recv")
            dur = summary.get("metrics", {}).get("iteration_duration", {})
            dur_ms = dur.get("values", {}).get("avg", 1000)
            scenario["type"] = "binary"
            scenario["payload_size"] = name.split("_")[1]
            scenario["bytes_recv"] = recv
            scenario["duration_s"] = round(dur_ms / 1000, 2)
            scenario["mb_per_sec"] = round((recv / (1024 * 1024)) / (dur_ms / 1000), 2) if dur_ms > 0 else 0

        elif name == "backpressure":
            recv = extract_counter_value(summary, "ws_bp_msgs_recv")
            sent = extract_counter_value(summary, "ws_bp_msgs_sent")
            scenario["type"] = "backpressure"
            scenario["msgs_sent"] = sent
            scenario["msgs_recv"] = recv
            scenario["delivery_ratio"] = round(recv / sent, 4) if sent > 0 else 0

        data["scenarios"][name] = scenario

        # Load server metrics
        server_path = os.path.join(results_dir, f"{name}_server.ndjson")
        server = load_server_metrics(server_path)
        if server:
            # Normalize timestamps to relative seconds
            t0 = server[0]["ts"] if server else 0
            data["server_metrics"][name] = [
                {"t": round((m["ts"] - t0) / 1000, 1), "rss": m["rss_kb"], "fds": m["fds"]}
                for m in server
            ]

    return data


HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>websocketd Benchmark Report</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: #f5f5f5; color: #333; padding: 20px; max-width: 1200px; margin: 0 auto; }
  h1 { margin-bottom: 4px; }
  .meta { color: #666; margin-bottom: 24px; font-size: 14px; }
  h2 { margin: 24px 0 12px; border-bottom: 2px solid #ddd; padding-bottom: 4px; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 20px; }
  .card { background: white; border-radius: 8px; padding: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  .card h3 { font-size: 14px; color: #666; margin-bottom: 8px; }
  .card canvas { width: 100% !important; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px;
          overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #eee; }
  th { background: #f8f8f8; font-weight: 600; font-size: 13px; color: #666; }
  td { font-variant-numeric: tabular-nums; }
  .num { text-align: right; font-family: 'SF Mono', Monaco, monospace; font-size: 13px; }
  @media (max-width: 768px) { .grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>

<h1>websocketd Benchmark Report</h1>
<div class="meta" id="meta"></div>

<h2>Summary</h2>
<table id="summary-table">
  <thead><tr><th>Scenario</th><th>Key Metric</th><th class="num">Value</th><th>Unit</th></tr></thead>
  <tbody></tbody>
</table>

<h2>Latency</h2>
<div class="grid">
  <div class="card"><h3>Echo Latency (1 connection, sequential)</h3><canvas id="chart-latency"></canvas></div>
  <div class="card"><h3>Sustained Load Latency (50 connections)</h3><canvas id="chart-sustained-latency"></canvas></div>
</div>

<h2>Throughput</h2>
<div class="grid">
  <div class="card"><h3>Messages per Second</h3><canvas id="chart-throughput"></canvas></div>
  <div class="card"><h3>Connection Storm (cycle time by VU count)</h3><canvas id="chart-storm"></canvas></div>
</div>

<h2>Binary</h2>
<div class="grid">
  <div class="card"><h3>Binary Throughput (MB/s by payload size)</h3><canvas id="chart-binary"></canvas></div>
  <div class="card"><h3>Backpressure</h3><canvas id="chart-backpressure"></canvas></div>
</div>

<h2>Server Resources</h2>
<div class="grid" id="server-charts"></div>

<script>
const D = /* BENCH_DATA_JSON */;

// Meta
const meta = D.meta || {};
document.getElementById('meta').textContent =
  `Version: ${meta.version || '?'} | Commit: ${meta.git_hash || '?'} | ` +
  `Date: ${meta.timestamp || '?'} | OS: ${meta.os || '?'} ${meta.arch || ''}`;

// Summary table
const tbody = document.querySelector('#summary-table tbody');
function addRow(name, metric, value, unit) {
  const tr = document.createElement('tr');
  tr.innerHTML = `<td>${name}</td><td>${metric}</td><td class="num">${value}</td><td>${unit}</td>`;
  tbody.appendChild(tr);
}

const S = D.scenarios || {};
if (S.echo_latency) addRow('Echo Latency', 'p95 RTT', S.echo_latency.p95, 'ms');
if (S.echo_throughput) addRow('Echo Throughput', 'msgs/sec', S.echo_throughput.msgs_per_sec, 'msgs/sec');
if (S.connection_churn) addRow('Connection Churn', 'conn/sec', S.connection_churn.conns_per_sec, 'conn/sec');
if (S.sustained_load) addRow('Sustained Load', 'p95 RTT', S.sustained_load.p95, 'ms');
if (S.sustained_load) addRow('Sustained Load', 'total msgs', S.sustained_load.total_msgs, 'msgs');
Object.keys(S).filter(k => k.startsWith('connection_storm_')).forEach(k => {
  addRow(`Storm (${S[k].vus} VUs)`, 'p95 cycle', S[k].p95, 'ms');
});
Object.keys(S).filter(k => k.startsWith('binary_')).forEach(k => {
  addRow(`Binary ${S[k].payload_size}`, 'throughput', S[k].mb_per_sec, 'MB/s');
});
if (S.backpressure) addRow('Backpressure', 'delivery ratio', S.backpressure.delivery_ratio, '');

// Chart helpers
const COLORS = ['#4e79a7','#f28e2b','#e15759','#76b7b2','#59a14f','#edc948'];
function barChart(id, labels, datasets) {
  const el = document.getElementById(id);
  if (!el) return;
  new Chart(el, {
    type: 'bar',
    data: { labels, datasets },
    options: { responsive: true, plugins: { legend: { display: datasets.length > 1 } },
               scales: { y: { beginAtZero: true } } }
  });
}

// Latency chart
if (S.echo_latency) {
  barChart('chart-latency', ['p50','p95','p99','avg','max'], [{
    label: 'ms', data: [S.echo_latency.p50, S.echo_latency.p95, S.echo_latency.p99,
                         S.echo_latency.avg, S.echo_latency.max],
    backgroundColor: COLORS
  }]);
}

if (S.sustained_load) {
  barChart('chart-sustained-latency', ['p50','p95','p99'], [{
    label: 'ms', data: [S.sustained_load.p50, S.sustained_load.p95, S.sustained_load.p99],
    backgroundColor: COLORS
  }]);
}

// Throughput chart
{
  const labels = [], values = [];
  if (S.echo_throughput) { labels.push('Echo (1 conn)'); values.push(S.echo_throughput.msgs_per_sec); }
  if (S.connection_churn) { labels.push('Churn (conn/sec)'); values.push(S.connection_churn.conns_per_sec); }
  if (labels.length) barChart('chart-throughput', labels, [{
    label: 'per second', data: values, backgroundColor: COLORS
  }]);
}

// Storm chart
{
  const storms = Object.keys(S).filter(k => k.startsWith('connection_storm_')).sort();
  if (storms.length) {
    barChart('chart-storm',
      storms.map(k => `${S[k].vus} VUs`),
      [{ label: 'p95 cycle (ms)', data: storms.map(k => S[k].p95), backgroundColor: COLORS[0] },
       { label: 'avg cycle (ms)', data: storms.map(k => S[k].avg), backgroundColor: COLORS[1] }]
    );
  }
}

// Binary chart
{
  const bins = Object.keys(S).filter(k => k.startsWith('binary_')).sort();
  if (bins.length) {
    barChart('chart-binary',
      bins.map(k => S[k].payload_size),
      [{ label: 'MB/s', data: bins.map(k => S[k].mb_per_sec), backgroundColor: COLORS[0] }]
    );
  }
}

// Backpressure chart
if (S.backpressure) {
  barChart('chart-backpressure', ['Sent','Received'], [{
    label: 'messages', data: [S.backpressure.msgs_sent, S.backpressure.msgs_recv],
    backgroundColor: [COLORS[0], COLORS[1]]
  }]);
}

// Server resource charts
const serverDiv = document.getElementById('server-charts');
const SM = D.server_metrics || {};
Object.keys(SM).forEach(name => {
  const points = SM[name];
  if (!points || !points.length) return;

  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `<h3>${name} — RSS (KB)</h3><canvas id="srv-${name}"></canvas>`;
  serverDiv.appendChild(card);

  new Chart(document.getElementById(`srv-${name}`), {
    type: 'line',
    data: {
      labels: points.map(p => p.t + 's'),
      datasets: [{
        label: 'RSS (KB)', data: points.map(p => p.rss),
        borderColor: COLORS[0], fill: false, pointRadius: 0, tension: 0.3
      }]
    },
    options: { responsive: true, scales: { y: { beginAtZero: true } },
               plugins: { legend: { display: false } } }
  });
});
</script>
</body>
</html>"""


def main():
    if len(sys.argv) < 2:
        print("Usage: generate-report.py <results-dir>", file=sys.stderr)
        sys.exit(1)

    results_dir = sys.argv[1]
    data = build_report_data(results_dir)

    html = HTML_TEMPLATE.replace("/* BENCH_DATA_JSON */", json.dumps(data))
    output_path = os.path.join(results_dir, "report.html")
    with open(output_path, "w") as f:
        f.write(html)

    print(f"  Wrote {output_path}")


if __name__ == "__main__":
    main()
