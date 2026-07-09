# Codebase Scorecard: websocketd

**Audited**: 2026-07-09 | **Size**: ~2.4 KLOC src + ~4.1 KLOC test | **Language**: Go | **Deps**: 1 (gorilla/websocket v1.5.3, pinned)

This audit re-verified every claim of the 2026-04-26 scorecard against the code,
CI history, and issue tracker, found drift in several areas, and fixed what it
found in the same session. Grades below reflect the state *after* those fixes;
the "found" column records what the audit walked into.

| # | Category | Found | Now | Key Finding |
|---|----------|-------|-----|-------------|
| 1 | Architecture | A- | A- | Clean handler chain, bidirectional PipeEndpoints, decomposed config validators — holds up |
| 2 | Code Quality | B+ | A- | Two goroutine leaks in endpoint readers (fixed); dead code removed |
| 3 | Consistency | B+ | A | gofmt drift in 3 files, missing license header, snake_case helper (all fixed, gofmt now gated in CI) |
| 4 | Security | A- | A- | Posture genuinely strong (origin checks, env whitelist, symlink boundary, mTLS); the one "injection" failure was a test artifact |
| 5 | Testing | A- | A | 217 tests verified accurate; brittle root-env assertion, 1.5s hardcoded sleep, and a dead harness capture all fixed |
| 6 | CI | B | A- | Tests green on 4 platforms with -race; Benchmarks workflow was red on every run since inception (fixed); linters now pinned, gofmt gated |
| 7 | Documentation | B- | B+ | --pingms/--sslca were absent from --help (fixed); man page was 12 years stale (rewritten); tutorial #438 and proxy docs #27 remain open |
| 8 | Release | D | B+ | Built 0.4.x labeled MIT with a Go that can't compile the module — all repaired; deb/rpm path untested against a real fpm install |
| 9 | Process hygiene | B- | B+ | DIARY/SCORECARD had been abandoned and counts inflated (84 vs actual 111 integration tests); corrected |

**Overall: A-** (honest A- this time: every grade above is backed by a
verified check, not aspiration)

## What this audit fixed

- **Goroutine leaks** (process_endpoint.go, websocket_endpoint.go): readers
  parked on the unbuffered output-channel send were never unblocked by
  Terminate, stranding up to 10MB per broken binary-mode connection.
  Regression test added (TestTerminateUnblocksParkedReader).
- **`go test ./...` failed as root**: TestSEC011 asserted no "root" substring
  in an env dump; PATH containing /root tripped it. Now asserts on output
  shape (env-assignment lines only).
- **Benchmarks CI red since creation**: k6 install relied on a flaky
  keyserver; now a pinned release binary. run.sh also discarded k6's exit
  code through a pipe and aborted the whole run when one server failed to
  start (both fixed). Regression alerts are advisory, not a merge gate.
- **Release tooling**: version 0.4→0.5, MIT→BSD-2-Clause package label,
  vendored Go 1.11/1.15 removed (cannot build a go 1.21 module), man page
  installed under its real name, bash required for brace-expansion recipes,
  man page rewritten with the 9 missing flags and correct defaults.
- **Docs**: --pingms and --sslca added to --help.
- **Test harness**: websocketd logs go to stdout, but the harness captured
  only stderr — two tests silently asserted nothing. Harness now captures
  both; those tests are real (and the dead-connection test polls instead of
  sleeping 1.5s).
- **Counts**: "84 integration tests" was wrong (111 measured; 141 top-level
  test functions + 76 subtests = 217 total). QA plans contain 321 case IDs,
  not ~270.

## Remaining issues

1. **LOW [Docs]** Tutorial rewrite (#438) and reverse-proxy docs (#27) still open.
2. **LOW [Test]** websocket_endpoint.go, env.go, and console.go still have no
   dedicated unit tests (integration-only coverage); process_endpoint.go now
   has one.
3. **LOW [Bench]** k6 scenarios use the deprecated `k6/ws` module; fine on the
   pinned k6 v1.0.0, will need migration to `k6/net/websockets` eventually.
   collect-metrics.sh samples only the parent PID, undercounting per-connection
   child processes.
4. **LOW [Test]** QA plans (qa/plans/) have no traceability mapping to the
   automated suite.
5. **LOW [Feature]** 9 open issues, no bugs. Most credible: --cgidir inside
   --staticdir serves CGI source as plain text (#453), stderr streaming (#403),
   config file (#350), frame size control (#445).

## Architecture assessment (unchanged)

ServeHTTP delegates to focused handlers; PipeEndpoints runs each direction
independently with natural backpressure; config parsing is decomposed into
pure, tested functions. No plugin system — the right tradeoff at this scope.
Handler coupling (WebsocketdHandler storing *WebsocketdServer) remains the
only structural nit.
