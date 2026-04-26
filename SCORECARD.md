# Codebase Scorecard: websocketd

**Audited**: 2026-04-26 | **Size**: 38 files, 6.4 KLOC (2.4K src + 4.0K test) | **Language**: Go | **Deps**: 1 (gorilla/websocket v1.5.3, pinned)

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | A- | Clean handler chain, bidirectional PipeEndpoints, extracted config validators |
| 2 | Code Quality | A- | No known bugs; one minor logic issue in readFrames type mismatch handling |
| 3 | Consistency | A- | All naming idiomatic Go; error vars ErrFoo, methods camelCase, uniform patterns |
| 4 | Security | A | Symlink boundary, origin hardened, env isolated, mTLS, gosec + staticcheck in CI |
| 5 | Performance | B+ | Regex compiled once, template cached at init, backpressure via OS pipes |
| 6 | DRY | B+ | Signal loop extracted; no meaningful duplication remains |
| 7 | Testability | A- | Extracted pure functions, Endpoint interface, per-test server isolation |
| 8 | Test Coverage | A | 217 tests (1.66:1 ratio), CI on 4 platforms with -race, staticcheck, gosec |
| 9 | Type Safety | A- | Go types used correctly throughout |
| 10 | Documentation | B | README accurate, CHANGES current; tutorial needs rewrite (#438) |
| 11 | Error Handling | A- | Errors returned not panicked, graceful fork tracking, gosec clean |
| 12 | Extensibility | B | Handler chain extensible, config validators composable |
| 13 | Repo Hygiene | A | Clean atomic commits, pinned deps, CI on 4 platforms, -race + lint + security scan |

**Overall: A-**

**Score history: B- (Apr 25) → B+ (Apr 26 AM) → A- (Apr 26 PM) → A- (Apr 26 EVE, 4 categories improved)**

## Top Strengths

- **PipeEndpoints** (endpoint.go:24-56) — bidirectional relay with independent goroutines, natural backpressure, zero application-level buffering
- **Test suite** — 217 tests across unit + integration, CI on Linux x86/ARM64 + macOS ARM64 + Windows x86_64, regression tests for 6 historical bugs
- **Security posture** — symlink boundary check, origin validation, env whitelist, mTLS (--sslca), ping/pong dead connection detection (--pingms), gosec + staticcheck clean
- **CI pipeline** — 4-platform testing with race detector, staticcheck linter, gosec security scanner; all green
- **Decomposed architecture** — ServeHTTP is a 20-line handler chain; config parsing delegates to 7 pure testable functions; no god objects

## Improvements This Round

- Consistency B+ → A-: Renamed error vars to ErrFoo convention, env.go replacers to idiomatic names
- Security A- → A: Added staticcheck + gosec to CI, preventing regressions
- Performance B → B+: Cached dev console template license substitution at init
- Repo Hygiene A- → A: Added -race detector, staticcheck, gosec to CI lint job

## Remaining Issues

1. **LOW [Docs]** Tutorial needs rewrite (#438) — detailed user feedback available.
2. **LOW [Docs]** Reverse proxy documentation (#27) — wiki page may exist but isn't linked from README.
3. **LOW [Feature]** 8 open issues, all deferred (feature requests and docs, no bugs).
4. **LOW [Code]** websocket_endpoint.go:113 readFrames logs "Ignoring" unexpected message type but still processes it.
5. **LOW [Code]** launcher.go:24-38 pipe file descriptors not explicitly closed on partial setup failure.
6. **LOW [Test]** 811 lines of endpoint/env code only covered by integration tests, no isolated unit tests.
7. **LOW [Test]** Several integration tests use hardcoded time.Sleep instead of polling patterns.

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
3. **Fix readFrames type-mismatch logic** — log + continue instead of processing. Code Quality A- → A.
4. **Close pipe FDs on launcher partial failure** — add deferred cleanup. Code Quality reinforcement.
5. **Replace time.Sleep in tests with polling** — reduces flakiness risk. Test Coverage robustness.
