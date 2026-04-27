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
            dur_v = dur.get("values", dur)
            dur_ms = dur_v.get("avg", 1000)
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


HTML_TEMPLATE = r"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>websocketd Benchmark Report</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: #fafafa; color: #222; padding: 24px; max-width: 960px; margin: 0 auto;
         line-height: 1.5; }
  h1 { font-size: 22px; font-weight: 600; margin-bottom: 2px; }
  .meta { font-size: 12px; color: #666; margin-bottom: 28px; line-height: 1.8; }
  .meta strong { color: #444; }
  h2 { font-size: 15px; font-weight: 600; color: #555; margin: 32px 0 16px;
       text-transform: uppercase; letter-spacing: 0.5px; }

  /* Key numbers */
  .kpi-row { display: flex; gap: 16px; margin-bottom: 28px; flex-wrap: wrap; }
  .kpi { flex: 1; min-width: 180px; background: white; border-radius: 6px; padding: 16px 20px;
         border: 1px solid #e8e8e8; }
  .kpi .label { font-size: 12px; color: #888; margin-bottom: 4px; }
  .kpi .value { font-size: 28px; font-weight: 600; font-variant-numeric: tabular-nums;
                font-family: 'SF Mono', Monaco, Consolas, monospace; color: #222; }
  .kpi .unit { font-size: 13px; color: #888; font-weight: 400; }
  .kpi .note { font-size: 11px; color: #aaa; margin-top: 2px; }

  /* Latency distribution */
  .dist { background: white; border-radius: 6px; padding: 20px; border: 1px solid #e8e8e8;
          margin-bottom: 16px; }
  .dist h3 { font-size: 13px; font-weight: 600; color: #444; margin-bottom: 2px; }
  .dist .desc { font-size: 12px; color: #999; margin-bottom: 14px; }
  .dist-row { display: flex; align-items: center; margin-bottom: 6px; font-size: 13px; }
  .dist-label { width: 36px; text-align: right; color: #888; font-size: 12px; margin-right: 10px; }
  .dist-bar-bg { flex: 1; height: 14px; background: #f0f0f0; border-radius: 2px; position: relative; }
  .dist-bar { height: 100%; background: #4e79a7; border-radius: 2px; min-width: 1px; }
  .dist-val { width: 70px; text-align: right; font-family: 'SF Mono', Monaco, monospace;
              font-size: 12px; color: #444; margin-left: 8px; }

  /* Charts */
  .chart-section { display: flex; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
  .chart-card { flex: 1; min-width: 300px; background: white; border-radius: 6px; padding: 16px;
                border: 1px solid #e8e8e8; }
  .chart-card h3 { font-size: 13px; font-weight: 600; color: #444; margin-bottom: 2px; }
  .chart-card .desc { font-size: 12px; color: #999; margin-bottom: 12px; }

  /* Backpressure special */
  .bp-stat { display: flex; align-items: baseline; gap: 8px; margin-bottom: 6px; }
  .bp-stat .bp-num { font-size: 24px; font-weight: 600; font-family: 'SF Mono', Monaco, monospace; }
  .bp-stat .bp-label { font-size: 13px; color: #666; }
  .bp-bar-bg { height: 20px; background: #f0f0f0; border-radius: 3px; position: relative; margin: 8px 0; }
  .bp-bar { height: 100%; background: #4e79a7; border-radius: 3px; }
  .bp-bar-text { font-size: 11px; color: #888; margin-top: 2px; }

  /* Server resources */
  .srv-card { background: white; border-radius: 6px; padding: 16px; border: 1px solid #e8e8e8;
              margin-bottom: 12px; }
  .srv-card h3 { font-size: 13px; font-weight: 600; color: #444; margin-bottom: 2px; }
  .srv-card .desc { font-size: 12px; color: #999; margin-bottom: 10px; }

  @media (max-width: 640px) { .kpi-row { flex-direction: column; } .chart-section { flex-direction: column; } }
</style>
</head>
<body>

<h1>websocketd Benchmark Report</h1>
<div class="meta" id="meta"></div>

<h2>Key Numbers</h2>
<div class="kpi-row" id="kpis"></div>

<h2>Latency Distribution</h2>
<div id="latency-dists"></div>

<h2>Scaling</h2>
<div class="chart-section" id="scaling-charts"></div>

<h2>Binary Throughput</h2>
<div class="chart-section" id="binary-section"></div>

<h2>Backpressure</h2>
<div id="bp-section"></div>

<h2>Memory (RSS)</h2>
<p style="font-size:12px;color:#999;margin-bottom:12px;">
  Resident memory of the websocketd process during each scenario. Only scenarios with meaningful
  duration are shown. Lower is better.
</p>
<div id="server-charts"></div>

<script>
const D = /* BENCH_DATA_JSON */;
const S = D.scenarios || {};
const SM = D.server_metrics || {};
const C = '#4e79a7';
const C2 = '#b8cfe0';

// --- Meta ---
const m = D.meta || {};
document.getElementById('meta').innerHTML = [
  `<strong>Version:</strong> ${m.version || '?'}`,
  `<strong>Commit:</strong> ${m.git_hash || '?'}`,
  `<strong>Date:</strong> ${m.timestamp || '?'}`,
  `<br><strong>OS:</strong> ${m.os_pretty || m.os || '?'} (${m.arch || '?'})`,
  `<strong>CPU:</strong> ${m.cpu || '?'} (${m.cpu_cores || '?'} cores)`,
  `<strong>RAM:</strong> ${m.ram_gb || '?'} GB`,
].join(' &nbsp;&middot;&nbsp; ');

// --- KPIs ---
const kpiDiv = document.getElementById('kpis');
function kpi(label, value, unit, note) {
  const d = document.createElement('div');
  d.className = 'kpi';
  d.innerHTML = `<div class="label">${label}</div>`
    + `<div class="value">${value} <span class="unit">${unit}</span></div>`
    + `<div class="note">${note}</div>`;
  kpiDiv.appendChild(d);
}
if (S.echo_latency) kpi('Echo Latency (p95)', S.echo_latency.p95, 'ms',
  '1 connection, 1000 sequential round-trips &middot; lower is better');
if (S.echo_throughput) kpi('Echo Throughput', Number(S.echo_throughput.msgs_per_sec).toLocaleString(), 'msgs/sec',
  '1 connection, 10s fire-hose &middot; higher is better');
if (S.connection_churn) kpi('Connection Churn', S.connection_churn.conns_per_sec, 'conn/sec',
  '200 serial connect/send/close cycles &middot; higher is better');
if (S.sustained_load) kpi('Sustained Load (p95)', S.sustained_load.p95, 'ms',
  '50 connections, 30s continuous &middot; lower is better');

// --- Latency distributions ---
const distDiv = document.getElementById('latency-dists');
function latencyDist(title, desc, percentiles) {
  // percentiles: [{label, value}] — value in ms
  const maxVal = Math.max(...percentiles.map(p => p.value), 0.001);
  const card = document.createElement('div');
  card.className = 'dist';
  let html = `<h3>${title}</h3><div class="desc">${desc}</div>`;
  for (const p of percentiles) {
    const pct = Math.max((p.value / maxVal) * 100, 0.5);
    html += `<div class="dist-row">`
      + `<div class="dist-label">${p.label}</div>`
      + `<div class="dist-bar-bg"><div class="dist-bar" style="width:${pct}%"></div></div>`
      + `<div class="dist-val">${p.value} ms</div></div>`;
  }
  card.innerHTML = html;
  distDiv.appendChild(card);
}
if (S.echo_latency) {
  latencyDist('Echo Latency',
    'Single connection, 1000 sequential message round-trips. Each bar shows where that percentile falls on the distribution. Lower is better.',
    [{label:'p50', value:S.echo_latency.p50}, {label:'p95', value:S.echo_latency.p95},
     {label:'p99', value:S.echo_latency.p99}, {label:'max', value:S.echo_latency.max}]);
}
if (S.sustained_load) {
  latencyDist('Sustained Load Latency',
    '50 concurrent connections sending continuously for 30 seconds. Measures latency under realistic multi-client load. Lower is better.',
    [{label:'p50', value:S.sustained_load.p50}, {label:'p95', value:S.sustained_load.p95},
     {label:'p99', value:S.sustained_load.p99}]);
}

// --- Scaling charts (storm + churn) ---
const scalingDiv = document.getElementById('scaling-charts');

// Storm: sorted numerically
const storms = Object.keys(S).filter(k => k.startsWith('connection_storm_'))
  .sort((a,b) => S[a].vus - S[b].vus);
if (storms.length) {
  const card = document.createElement('div');
  card.className = 'chart-card';
  card.innerHTML = `<h3>Connection Storm</h3>`
    + `<div class="desc">N clients connect simultaneously, each sends one message and disconnects. `
    + `Shows how cycle time grows with concurrency. Lower is better.</div>`
    + `<canvas id="chart-storm"></canvas>`;
  scalingDiv.appendChild(card);

  new Chart(document.getElementById('chart-storm'), {
    type: 'bar',
    data: {
      labels: storms.map(k => S[k].vus + ' connections'),
      datasets: [
        { label: 'p95', data: storms.map(k => S[k].p95), backgroundColor: C },
        { label: 'avg', data: storms.map(k => S[k].avg), backgroundColor: C2 }
      ]
    },
    options: {
      responsive: true,
      plugins: { legend: { labels: { boxWidth: 12, font: { size: 11 } } } },
      scales: {
        y: { beginAtZero: true, title: { display: true, text: 'Cycle time (ms)', font: { size: 11 } },
             grid: { color: '#f0f0f0' } },
        x: { grid: { display: false } }
      }
    }
  });
}

// --- Binary throughput ---
const binaryDiv = document.getElementById('binary-section');
const bins = Object.keys(S).filter(k => k.startsWith('binary_'))
  .sort((a,b) => {
    const sizeOrder = {'1k':1,'10k':2,'64k':3,'256k':4};
    return (sizeOrder[S[a].payload_size]||99) - (sizeOrder[S[b].payload_size]||99);
  });
if (bins.length) {
  const card = document.createElement('div');
  card.className = 'chart-card';
  card.innerHTML = `<h3>Binary Mode Throughput</h3>`
    + `<div class="desc">websocketd running with --binary flag. 100 round-trips per payload size `
    + `using a cat backend. Payload sizes sorted smallest to largest. Higher is better.</div>`
    + `<canvas id="chart-binary"></canvas>`;
  binaryDiv.appendChild(card);

  new Chart(document.getElementById('chart-binary'), {
    type: 'bar',
    data: {
      labels: bins.map(k => S[k].payload_size.toUpperCase()),
      datasets: [{ label: 'MB/s', data: bins.map(k => S[k].mb_per_sec), backgroundColor: C }]
    },
    options: {
      responsive: true,
      plugins: { legend: { display: false } },
      scales: {
        y: { beginAtZero: true, title: { display: true, text: 'Throughput (MB/s)', font: { size: 11 } },
             grid: { color: '#f0f0f0' } },
        x: { grid: { display: false } }
      }
    }
  });
}

// --- Backpressure ---
const bpDiv = document.getElementById('bp-section');
if (S.backpressure) {
  const bp = S.backpressure;
  const pct = (bp.delivery_ratio * 100).toFixed(1);
  const card = document.createElement('div');
  card.className = 'dist';
  card.innerHTML = `<h3>Backpressure Test</h3>`
    + `<div class="desc">Client sends messages as fast as possible to a backend that processes `
    + `one message every 100ms. websocketd's pipe-based backpressure limits how many messages `
    + `actually reach the backend. This is correct behavior, not a bug.</div>`
    + `<div class="bp-stat"><span class="bp-num">${bp.msgs_sent.toLocaleString()}</span>`
    + `<span class="bp-label">messages sent by client</span></div>`
    + `<div class="bp-stat"><span class="bp-num">${bp.msgs_recv.toLocaleString()}</span>`
    + `<span class="bp-label">messages echoed by backend (${pct}%)</span></div>`
    + `<div class="bp-bar-bg"><div class="bp-bar" style="width:${Math.max(bp.delivery_ratio*100,0.5)}%"></div></div>`
    + `<div class="bp-bar-text">${pct}% delivery ratio &mdash; the rest were absorbed by OS pipe backpressure</div>`;
  bpDiv.appendChild(card);
}

// --- Server resources ---
// Only show scenarios that ran long enough to have interesting data (>3 data points)
const serverDiv = document.getElementById('server-charts');
const NAMES = {
  echo_latency: 'Echo Latency', echo_throughput: 'Echo Throughput',
  connection_churn: 'Connection Churn', sustained_load: 'Sustained Load (50 conns, 30s)',
  backpressure: 'Backpressure (15s)',
};
function srvName(n) {
  if (NAMES[n]) return NAMES[n];
  if (n.startsWith('connection_storm_')) return 'Storm (' + n.split('_').pop() + ' conns)';
  if (n.startsWith('binary_')) return 'Binary ' + n.split('_').pop().toUpperCase();
  return n;
}

// Filter to only scenarios with >3 data points (short ones are uninteresting flat lines)
const interestingSrv = Object.keys(SM).filter(n => SM[n] && SM[n].length > 3);
for (const name of interestingSrv) {
  const points = SM[name];
  const card = document.createElement('div');
  card.className = 'srv-card';
  card.innerHTML = `<h3>${srvName(name)}</h3>`
    + `<div class="desc">Peak: ${Math.max(...points.map(p=>p.rss)).toLocaleString()} KB &middot; Lower is better</div>`
    + `<canvas id="srv-${name}"></canvas>`;
  serverDiv.appendChild(card);

  new Chart(document.getElementById(`srv-${name}`), {
    type: 'line',
    data: {
      labels: points.map(p => p.t + 's'),
      datasets: [{
        data: points.map(p => p.rss),
        borderColor: C, borderWidth: 1.5, fill: true, backgroundColor: C + '18',
        pointRadius: 0, tension: 0.3
      }]
    },
    options: {
      responsive: true, aspectRatio: 3,
      plugins: { legend: { display: false } },
      scales: {
        y: { beginAtZero: true, title: { display: true, text: 'RSS (KB)', font: { size: 11 } },
             grid: { color: '#f0f0f0' } },
        x: { grid: { display: false }, ticks: { maxTicksLimit: 10, font: { size: 10 } } }
      }
    }
  });
}
if (!interestingSrv.length) {
  serverDiv.innerHTML = '<p style="color:#999;font-size:13px;">No scenarios ran long enough to capture meaningful memory data.</p>';
}
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
