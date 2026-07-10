window.BENCHMARK_DATA = {
  "lastUpdate": 1783643792238,
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
          "id": "e40012e5ed06bd49c167ee7534540d9f61bc85e1",
          "message": "Add --unixsocket to listen on a Unix domain socket\n\n* Add --unixsocket to listen on a Unix domain socket\n\nRebase and rework of #435 (by @matvore), renamed from --uds to\n--unixsocket to match this codebase's flag naming (no hyphens:\n--staticdir, --cgidir, --reverselookup, etc).\n\nServes alongside --address/--port by default; given alone (no --port,\n--address, or --redirport), no TCP listener is started at all - for\nexposing websocketd only to processes on the same host, e.g. behind\nan SSH-forwarded or reverse-proxied socket.\n\nChanges from the original PR:\n- TCP and Unix listeners now share one serve() helper (also handling\n  plain HTTP, TLS, and mutual TLS) instead of duplicating the\n  Ssl/SslCaFile branch inline; avoids a merge conflict with the\n  mutual-TLS support added after this PR was opened\n- GetRemoteInfo returns a stable unix-socket placeholder instead of\n  erroring on a Unix peer address (which has no host:port to parse),\n  fixed in the shared function itself rather than special-cased at\n  the handler.go call site, so every caller benefits\n- A stale socket file left behind by an unclean shutdown (a killed\n  process never gets to unlink it) is now removed automatically\n  before binding, rather than failing with address already in use\n- Added unit tests (wantsUnixSocketOnly, GetRemoteInfo) and two\n  integration tests (echo round-trip over a real socket, stale-socket\n  recovery), plus a QA plan entry and docs (--help, README, man page)\n\nVerified manually end-to-end: real WebSocket handshake + echo over a\nUnix socket, unixsocket-only mode confirmed to skip the TCP listener,\nand stale-socket cleanup after a simulated SIGKILL crash.\n\nCo-Authored-By: matvore <matvore@users.noreply.github.com>\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_01M882UWfvyaq5KGvaV37idr\n\n* Fix UDS integration tests on macOS: sun_path 104-byte limit\n\nt.TempDir() on macOS CI nests under a long $TMPDIR plus the test\nname and a /001/ subdir, routinely exceeding sockaddr_un.sun_path's\n104-byte limit (108 on Linux) and failing bind with EINVAL. Use a\nshort path under /tmp directly instead.\n\nCo-Authored-By: Claude Fable 5 <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_01M882UWfvyaq5KGvaV37idr\n\n---------\n\nCo-authored-by: Claude <noreply@anthropic.com>\nCo-authored-by: matvore <matvore@users.noreply.github.com>",
          "timestamp": "2026-07-09T17:09:25-07:00",
          "tree_id": "11d0d9bcc644f022261aa4ecf82da8cb5d35bb93",
          "url": "https://github.com/joewalnes/websocketd/commit/e40012e5ed06bd49c167ee7534540d9f61bc85e1"
        },
        "date": 1783642247798,
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
            "value": 12976,
            "unit": "KB"
          },
          {
            "name": "binary_10k_MB_sec",
            "value": 0.98,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_10k_peak_rss_kb",
            "value": 13508,
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
            "value": 13456,
            "unit": "KB"
          },
          {
            "name": "connection_churn_avg_ms",
            "value": 1.5,
            "unit": "ms"
          },
          {
            "name": "connection_churn_conns_sec",
            "value": 666.7,
            "unit": "conn/sec (info only)"
          },
          {
            "name": "connection_churn_peak_rss_kb",
            "value": 10872,
            "unit": "KB"
          },
          {
            "name": "connection_storm_100_p95",
            "value": 66,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_avg",
            "value": 45.46,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_peak_rss_kb",
            "value": 8552,
            "unit": "KB"
          },
          {
            "name": "connection_storm_10_p95",
            "value": 8.55,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_avg",
            "value": 7,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_peak_rss_kb",
            "value": 10872,
            "unit": "KB"
          },
          {
            "name": "connection_storm_500_p95",
            "value": 370,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_avg",
            "value": 286.768,
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
            "value": 0.14,
            "unit": "ms"
          },
          {
            "name": "echo_latency_peak_rss_kb",
            "value": 10840,
            "unit": "KB"
          },
          {
            "name": "echo_throughput_us_per_msg",
            "value": 30.664,
            "unit": "µs/msg"
          },
          {
            "name": "echo_throughput_msgs_sec",
            "value": 32611,
            "unit": "msgs/sec (info only)"
          },
          {
            "name": "echo_throughput_peak_rss_kb",
            "value": 15412,
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
            "value": 16120,
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
          "id": "16d3db0fe81e4346d18fce8949df3d12f0539db4",
          "message": "Add --passstderr to forward STDERR to WebSocket clients as tagged JSON\n\nRebase and rework of #459 (by @Formatted) onto current master.\n\nForwards STDERR to WebSocket clients as tagged JSON, alongside tagged\nSTDOUT, so a client can tell the two apart:\n  {\"stream\":\"stdout\",\"data\":\"...\"}\n  {\"stream\":\"stderr\",\"data\":\"...\"}\nSTDERR is still logged server-side either way, same as without the\nflag. Addresses #403 (open since 2021).\n\nChanges from the original PR:\n- The tagged stdout/stderr readers now integrate with the done-channel\n  leak fix from the earlier goroutine-leak PR: each select{}s on the\n  output send against Terminate's done signal, same as the plain text\n  and binary readers, instead of blocking unconditionally. Verified\n  by temporarily reverting just that part and watching the new\n  regression test fail (3 leaked goroutines), then restoring it.\n- --binary and --passstderr are now mutually exclusive, rejected at\n  startup with a clear error. The original PR silently dropped\n  --binary whenever --passstderr was set (StartReading branched on\n  passStderr before bin), which would corrupt binary output instead\n  of erroring - tagging arbitrary binary chunks as JSON string data\n  isn't implemented, so refusing the combination is safer than a\n  partial implementation.\n- JSON encoding now goes through encoding/json (a small taggedMessage\n  struct) instead of a hand-rolled escaper, so it can't emit invalid\n  JSON for control characters or non-UTF8 bytes the original escaper\n  didn't handle.\n- Added a --binary/--passstderr validation unit test, a goroutine-leak\n  regression test mirroring the process-endpoint one, an integration\n  test asserting the tagged JSON over a real WebSocket connection (and\n  that STDERR still reaches the server log), and a QA plan entry.\n\n\n\nClaude-Session: https://claude.ai/code/session_01M882UWfvyaq5KGvaV37idr\n\nCo-authored-by: Claude <noreply@anthropic.com>\nCo-authored-by: Formatted <14853553+Formatted@users.noreply.github.com>",
          "timestamp": "2026-07-09T17:35:10-07:00",
          "tree_id": "cfeaf6e01dd4d51a979499956894217b7fc7d88b",
          "url": "https://github.com/joewalnes/websocketd/commit/16d3db0fe81e4346d18fce8949df3d12f0539db4"
        },
        "date": 1783643791958,
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
            "value": 10952,
            "unit": "KB"
          },
          {
            "name": "binary_10k_MB_sec",
            "value": 0.98,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_10k_peak_rss_kb",
            "value": 13620,
            "unit": "KB"
          },
          {
            "name": "binary_1k_MB_sec",
            "value": 0.1,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_1k_peak_rss_kb",
            "value": 13552,
            "unit": "KB"
          },
          {
            "name": "binary_64k_MB_sec",
            "value": 6.25,
            "unit": "MB/s (info only)"
          },
          {
            "name": "binary_64k_peak_rss_kb",
            "value": 13620,
            "unit": "KB"
          },
          {
            "name": "connection_churn_avg_ms",
            "value": 1.6,
            "unit": "ms"
          },
          {
            "name": "connection_churn_conns_sec",
            "value": 625,
            "unit": "conn/sec (info only)"
          },
          {
            "name": "connection_churn_peak_rss_kb",
            "value": 10964,
            "unit": "KB"
          },
          {
            "name": "connection_storm_100_p95",
            "value": 72.1,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_avg",
            "value": 51.39,
            "unit": "ms"
          },
          {
            "name": "connection_storm_100_peak_rss_kb",
            "value": 8580,
            "unit": "KB"
          },
          {
            "name": "connection_storm_10_p95",
            "value": 8,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_avg",
            "value": 6.2,
            "unit": "ms"
          },
          {
            "name": "connection_storm_10_peak_rss_kb",
            "value": 8588,
            "unit": "KB"
          },
          {
            "name": "connection_storm_500_p95",
            "value": 304.05,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_avg",
            "value": 175.692,
            "unit": "ms"
          },
          {
            "name": "connection_storm_500_peak_rss_kb",
            "value": 8576,
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
            "value": 0.139,
            "unit": "ms"
          },
          {
            "name": "echo_latency_peak_rss_kb",
            "value": 8580,
            "unit": "KB"
          },
          {
            "name": "echo_throughput_us_per_msg",
            "value": 29.466,
            "unit": "µs/msg"
          },
          {
            "name": "echo_throughput_msgs_sec",
            "value": 33937,
            "unit": "msgs/sec (info only)"
          },
          {
            "name": "echo_throughput_peak_rss_kb",
            "value": 15168,
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
            "value": 174951,
            "unit": "msgs (info only)"
          },
          {
            "name": "sustained_load_peak_rss_kb",
            "value": 16252,
            "unit": "KB"
          }
        ]
      }
    ]
  }
}