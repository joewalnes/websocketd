# Codebase Scorecard: websocketd

**Audited**: 2026-04-26 | **Size**: 38 files, 6.4 KLOC (2.4K src + 4.0K test) | **Language**: Go | **Deps**: 1 (gorilla/websocket v1.5.3, pinned)

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | A- | Clean handler chain, bidirectional PipeEndpoints, extracted config validators |
| 2 | Code Quality | A | No known bugs; readFrames, launcher, and all error paths fixed |
| 3 | Consistency | A- | All naming idiomatic Go; error vars ErrFoo, methods camelCase, uniform patterns |
| 4 | Security | A | Symlink boundary, origin hardened, env isolated, mTLS, gosec + staticcheck in CI |
| 5 | Performance | A- | Regex compiled once, template cached, strings.Builder in hot path, OS pipe backpressure |
| 6 | DRY | B+ | Signal loop extracted; no meaningful duplication remains |
| 7 | Testability | A- | Extracted pure functions, Endpoint interface, per-test server isolation |
| 8 | Test Coverage | A | 217 tests (1.66:1 ratio), CI on 4 platforms with -race, polling-based waits |
| 9 | Type Safety | A- | Go types used correctly throughout |
| 10 | Documentation | B | README accurate, CHANGES current; tutorial needs rewrite (#438) |
| 11 | Error Handling | A | Errors returned not panicked, pipe FDs cleaned up, gosec clean |
| 12 | Extensibility | B | Handler chain extensible, config validators composable |
| 13 | Repo Hygiene | A | Clean atomic commits, pinned deps, CI on 4 platforms, -race + lint + security scan |

**Overall: A-** (high A-, 6 categories at full A)

**Score history: B- (Apr 25) → B+ (Apr 26 AM) → A- (Apr 26 PM) → A- (Apr 26 EVE, 8 categories improved total)**

## Top Strengths

- **PipeEndpoints** (endpoint.go:24-56) — bidirectional relay with independent goroutines, natural backpressure, zero application-level buffering
- **Test suite** — 217 tests across unit + integration, CI on Linux x86/ARM64 + macOS ARM64 + Windows x86_64, regression tests for 6 historical bugs, polling-based waits
- **Security posture** — symlink boundary check, origin validation, env whitelist, mTLS (--sslca), ping/pong dead connection detection (--pingms), gosec + staticcheck clean
- **CI pipeline** — 4-platform testing with race detector, staticcheck linter, gosec security scanner; all green
- **Decomposed architecture** — ServeHTTP is a 20-line handler chain; config parsing delegates to 7 pure testable functions; no god objects

## Improvements This Session

| Category | Start | Now | Key Change |
|----------|-------|-----|------------|
| Code Quality | A- | A | Fixed readFrames type-mismatch, launcher pipe leak |
| Consistency | B+ | A- | Error vars ErrFoo, env.go replacers renamed |
| Security | A- | A | staticcheck + gosec in CI |
| Performance | B | A- | Template cached, strings.Builder in appendEnv |
| Error Handling | A- | A | Pipe FDs cleaned up on partial failure |
| Repo Hygiene | A- | A | -race, staticcheck, gosec in CI |
| Test Coverage | A | A | Polling replaces hardcoded sleeps |

## Remaining Issues

1. **LOW [Docs]** Tutorial needs rewrite (#438) — detailed user feedback available.
2. **LOW [Docs]** Reverse proxy documentation (#27) — wiki page may exist but isn't linked from README.
3. **LOW [Feature]** 8 open issues, all deferred (feature requests and docs, no bugs).
4. **LOW [Test]** Endpoint and env code (process_endpoint, websocket_endpoint, env.go) only covered by integration tests, no isolated unit tests.

## Architecture Assessment

The architecture is clean across all layers. ServeHTTP delegates to focused handlers. PipeEndpoints runs each direction independently. Config parsing is decomposed into testable functions. The Endpoint interface decouples WebSocket I/O from process I/O. Signal escalation is a single loop over a table.

The main limitation is scope-appropriate: no plugin/middleware system. Adding a new handler type still requires editing ServeHTTP. For a focused CLI tool this is the right tradeoff.

Handler coupling (WebsocketdHandler stores *WebsocketdServer) is a minor concern — injecting Config directly would improve testability, but the current design works well at this scale.

## Documentation vs Reality

- **Tutorial** (#438): User-reported confusion about step ordering and port configuration. Not yet addressed.
- **URLInfo**: Embedded in WebsocketdHandler but only FilePath and ScriptPath are used externally. Minor over-exposure.

## Quick Wins

1. **Rewrite the 10-minute tutorial** — detailed user feedback in #438. Documentation B → A-.
2. **Link the nginx wiki page from README** — closes #27. One line.
3. **Add unit tests for endpoint/env code** — increases isolated coverage. Testability A- → A.
