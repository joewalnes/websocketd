# Codebase Scorecard: websocketd

**Audited**: 2026-04-26 | **Size**: 27 files, 7.0 KLOC (3.1K src + 3.9K test) | **Language**: Go | **Deps**: 1 (gorilla/websocket v1.5.3, pinned)

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | A- | Clean handler chain, bidirectional PipeEndpoints, extracted config validators |
| 2 | Code Quality | A- | No known bugs; all production error paths handled |
| 3 | Consistency | B+ | All methods camelCase, uniform error-return pattern; some legacy style in env.go |
| 4 | Security | A- | Symlink boundary check, origin hardened, env isolated, mTLS, gosec clean |
| 5 | Performance | B | Regex compiled once, backpressure via OS pipes, no per-request allocations |
| 6 | DRY | B+ | Signal loop extracted; no meaningful duplication remains |
| 7 | Testability | A- | Extracted pure functions, Endpoint interface, per-test server isolation |
| 8 | Test Coverage | A | 217 tests (1.28:1 ratio), all production files unit tested, CI on 4 platforms |
| 9 | Type Safety | A- | Go types used correctly throughout |
| 10 | Documentation | B | README accurate, CHANGES current; tutorial needs rewrite (#438) |
| 11 | Error Handling | A- | Errors returned not panicked, graceful fork tracking, gosec clean |
| 12 | Extensibility | B | Handler chain extensible, config validators composable |
| 13 | Repo Hygiene | A- | Clean atomic commits, pinned deps, CI on 4 platforms, no junk |

**Overall: A-**

**Score history: B- (Apr 25) → B+ (Apr 26 AM) → A- (Apr 26 PM)**

## Top Strengths

- **PipeEndpoints** (endpoint.go:24-56) — bidirectional relay with independent goroutines, natural backpressure, zero application-level buffering
- **Test suite** — 217 tests across unit + integration, CI on Linux x86/ARM64 + macOS ARM64 + Windows x86_64, regression tests for 6 historical bugs
- **Security posture** — symlink boundary check, origin validation, env whitelist, mTLS (--sslca), ping/pong dead connection detection (--pingms), gosec clean
- **Decomposed architecture** — ServeHTTP is a 20-line handler chain; config parsing delegates to 7 pure testable functions; no god objects

## Remaining Issues

1. **LOW [Docs]** Tutorial needs rewrite (#438) — detailed user feedback available.
2. **LOW [Docs]** Reverse proxy documentation (#27) — wiki page may exist but isn't linked from README.
3. **LOW [Feature]** 8 open issues, all deferred (feature requests and docs, no bugs).

## Architecture Assessment

The architecture is clean across all layers. ServeHTTP delegates to focused handlers. PipeEndpoints runs each direction independently. Config parsing is decomposed into testable functions. The Endpoint interface decouples WebSocket I/O from process I/O. Signal escalation is a single loop over a table.

The main limitation is scope-appropriate: no plugin/middleware system. Adding a new handler type still requires editing ServeHTTP. For a focused CLI tool this is the right tradeoff.

## Documentation vs Reality

- **Tutorial** (#438): User-reported confusion about step ordering and port configuration. Not yet addressed.
- **URLInfo**: Embedded in WebsocketdHandler but only FilePath and ScriptPath are used externally. Minor over-exposure.

## Quick Wins

1. **Rewrite the 10-minute tutorial** — detailed user feedback in #438. Documentation B → A-.
2. **Link the nginx wiki page from README** — closes #27. One line.
3. **Add `go test -race` to CI** — catches test infrastructure races. One line in YAML.
