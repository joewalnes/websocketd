window.BENCHMARK_DATA = {
  "lastUpdate": 1783640829845,
  "repoUrl": "https://github.com/joewalnes/websocketd",
  "entries": {
    "websocketd Performance": [
      {
        "commit": {
          "author": {
            "email": "noreply@anthropic.com",
            "name": "Claude",
            "username": "claude"
          },
          "committer": {
            "email": "joe@walnes.com",
            "name": "Joe Walnes",
            "username": "joewalnes"
          },
          "distinct": true,
          "id": "8827e615aa21422c2ba34afc0316a9583069520d",
          "message": "Add leak regression test for the WebSocket endpoint\n\nThe done-channel fix landed in both endpoints but only ProcessEndpoint\nhad a regression test; the readFrames select path was covered only\nindirectly. Mirror the test with a real in-process WebSocket pair:\npark the reader by never draining Output(), Terminate, and require\nthe goroutine count to return to baseline. Verified it fails with the\nreadFrames fix reverted.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_01M882UWfvyaq5KGvaV37idr",
          "timestamp": "2026-07-09T16:30:41-07:00",
          "tree_id": "57fe5ab8ff09f263391701bc439b6b01174c8b10",
          "url": "https://github.com/joewalnes/websocketd/commit/8827e615aa21422c2ba34afc0316a9583069520d"
        },
        "date": 1783639936002,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "backpressure_msgs_echoed",
            "value": 149,
            "unit": "msgs (info only)"
          },
          {
            "name": "backpressure_delivery_ratio",
            "value": 0.0149,
            "unit": "ratio (info only)"
          },
          {
            "name": "backpressure_peak_rss_kb",
            "value": 12984,
            "unit": "KB"
          },
          {
            "name": "binary_10k_MB_sec",
            "value": 0.98,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_10k_peak_rss_kb",
            "value": 13500,
            "unit": "KB"
          },
          {
            "name": "binary_1k_MB_sec",
            "value": 0.1,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_1k_peak_rss_kb",
            "value": 13468,
            "unit": "KB"
          },
          {
            "name": "binary_64k_MB_sec",
            "value": 6.25,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_64k_peak_rss_kb",
            "value": 15640,
            "unit": "KB"
          },
          {
            "name": "connection_churn_avg_ms",
            "value": 1.52,
            "unit": "ms"
          },
          {
            "name": "connection_churn_conns_sec",
            "value": 657.9,
            "unit": "conn/sec (info only)"
          },
          {
            "name": "connection_churn_peak_rss_kb",
            "value": 8788,
            "unit": "KB"
          },
          {
            "name": "connection_storm_100_p95",
            "value": 63.25,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_avg",
            "value": 54.64,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_peak_rss_kb",
            "value": 8552,
            "unit": "KB"
          },
          {
            "name": "connection_storm_10_p95",
            "value": 10,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_avg",
            "value": 8.4,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_peak_rss_kb",
            "value": 10952,
            "unit": "KB"
          },
          {
            "name": "connection_storm_500_p95",
            "value": 305,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_avg",
            "value": 196.37,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_peak_rss_kb",
            "value": 8552,
            "unit": "KB"
          },
          {
            "name": "echo_latency_p50",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "echo_latency_p95",
            "value": 1,
            "unit": "ms"
          },
          {
            "name": "echo_latency_p99",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "echo_latency_avg",
            "value": 0.15,
            "unit": "ms"
          },
          {
            "name": "echo_latency_peak_rss_kb",
            "value": 8784,
            "unit": "KB"
          },
          {
            "name": "echo_throughput_us_per_msg",
            "value": 30.983,
            "unit": "µs/msg"
          },
          {
            "name": "echo_throughput_msgs_sec",
            "value": 32276,
            "unit": "msgs/sec (info only)"
          },
          {
            "name": "echo_throughput_peak_rss_kb",
            "value": 15544,
            "unit": "KB"
          },
          {
            "name": "sustained_load_rtt_p50",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "sustained_load_rtt_p95",
            "value": 1,
            "unit": "ms"
          },
          {
            "name": "sustained_load_rtt_p99",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "sustained_load_total_msgs",
            "value": 174950,
            "unit": "msgs (info only)"
          },
          {
            "name": "sustained_load_peak_rss_kb",
            "value": 16240,
            "unit": "KB"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "joe@walnes.com",
            "name": "Joe Walnes",
            "username": "joewalnes"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "5ec965a9648813aad10d537ec712156c46c29409",
          "message": "Add PowerShell examples (count, greeter, dump-env)\n\nMirrors the existing windows-vbscript/windows-jscript example\ndirectories, but PowerShell Core also runs on Linux/macOS via the\n#!/usr/bin/env pwsh shebang, so this isn't Windows-only like its\nsiblings. count.ps1's 10-iteration/500ms timing matches\nbash/count.sh; dump-env.ps1 reads the same CGI variable list as the\nother dump-env examples; greeter.ps1 only interpolates input into a\nWrite-Host format string (no eval), so there's no injection surface.\n\nMentions PowerShell in the README language list and adds a QA plan\nentry (WIN-007) alongside the existing VBScript/JScript ones.\n\nOriginally proposed in #423 by @kshahar (tested on Windows 10 /\nPowerShell 5.1 and Ubuntu 18.04 / PowerShell 7.2); rebased onto\ncurrent master and given a CHANGES entry.\n\n\n\nClaude-Session: https://claude.ai/code/session_01M882UWfvyaq5KGvaV37idr\n\nCo-authored-by: Claude <noreply@anthropic.com>\nCo-authored-by: Kobi Shahar <kshahar@users.noreply.github.com>",
          "timestamp": "2026-07-09T16:45:41-07:00",
          "tree_id": "fd892fa83df255c2a04ae25965ca7555144a12c7",
          "url": "https://github.com/joewalnes/websocketd/commit/5ec965a9648813aad10d537ec712156c46c29409"
        },
        "date": 1783640828854,
        "tool": "customSmallerIsBetter",
        "benches": [
          {
            "name": "backpressure_msgs_echoed",
            "value": 149,
            "unit": "msgs (info only)"
          },
          {
            "name": "backpressure_delivery_ratio",
            "value": 0.0149,
            "unit": "ratio (info only)"
          },
          {
            "name": "backpressure_peak_rss_kb",
            "value": 12980,
            "unit": "KB"
          },
          {
            "name": "binary_10k_MB_sec",
            "value": 0.98,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_10k_peak_rss_kb",
            "value": 13528,
            "unit": "KB"
          },
          {
            "name": "binary_1k_MB_sec",
            "value": 0.1,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_1k_peak_rss_kb",
            "value": 13464,
            "unit": "KB"
          },
          {
            "name": "binary_64k_MB_sec",
            "value": 6.25,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_64k_peak_rss_kb",
            "value": 13528,
            "unit": "KB"
          },
          {
            "name": "connection_churn_avg_ms",
            "value": 1.58,
            "unit": "ms"
          },
          {
            "name": "connection_churn_conns_sec",
            "value": 632.9,
            "unit": "conn/sec (info only)"
          },
          {
            "name": "connection_churn_peak_rss_kb",
            "value": 10856,
            "unit": "KB"
          },
          {
            "name": "connection_storm_100_p95",
            "value": 62.05,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_avg",
            "value": 41.22,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_peak_rss_kb",
            "value": 8548,
            "unit": "KB"
          },
          {
            "name": "connection_storm_10_p95",
            "value": 8,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_avg",
            "value": 6.7,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_peak_rss_kb",
            "value": 10920,
            "unit": "KB"
          },
          {
            "name": "connection_storm_500_p95",
            "value": 301.05,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_avg",
            "value": 178.456,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_peak_rss_kb",
            "value": 8548,
            "unit": "KB"
          },
          {
            "name": "echo_latency_p50",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "echo_latency_p95",
            "value": 1,
            "unit": "ms"
          },
          {
            "name": "echo_latency_p99",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "echo_latency_avg",
            "value": 0.13,
            "unit": "ms"
          },
          {
            "name": "echo_latency_peak_rss_kb",
            "value": 8772,
            "unit": "KB"
          },
          {
            "name": "echo_throughput_us_per_msg",
            "value": 26.11,
            "unit": "µs/msg"
          },
          {
            "name": "echo_throughput_msgs_sec",
            "value": 38299,
            "unit": "msgs/sec (info only)"
          },
          {
            "name": "echo_throughput_peak_rss_kb",
            "value": 15684,
            "unit": "KB"
          },
          {
            "name": "sustained_load_rtt_p50",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "sustained_load_rtt_p95",
            "value": 1,
            "unit": "ms"
          },
          {
            "name": "sustained_load_rtt_p99",
            "value": 0,
            "unit": "ms"
          },
          {
            "name": "sustained_load_total_msgs",
            "value": 174950,
            "unit": "msgs (info only)"
          },
          {
            "name": "sustained_load_peak_rss_kb",
            "value": 16096,
            "unit": "KB"
          }
        ]
      }
    ]
  }
}