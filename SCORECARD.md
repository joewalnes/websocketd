# Codebase Scorecard: websocketd

**Audited**: 2026-04-26 | **Size**: 27 files, 7.0 KLOC (3.1K src + 3.9K test) | **Language**: Go | **Deps**: 1 (gorilla/websocket v1.5.3, pinned)

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | A- | Clean handler chain, bidirectional PipeEndpoints, extracted config validators |
| 2 | Code Quality | B+ | No known bugs; defensive panic in fork tracking (http.go:252) |
| 3 | Consistency | B+ | All methods camelCase, uniform error-return pattern throughout |
| 4 | Security | A- | Symlink boundary check, origin hardened, env isolated, mTLS, gosec clean |
| 5 | Performance | B | Regex compiled once, backpressure via OS pipes, no per-request allocations |
| 6 | DRY | B+ | Terminate() signal loop extracted; no meaningful duplication remains |
| 7 | Testability | A- | 12 extracted pure functions, Endpoint interface, per-test server isolation |
| 8 | Test Coverage | A | 217 tests (1.28:1 test-to-code ratio), all production files have unit tests |
| 9 | Type Safety | A- | Go types used correctly throughout |
| 10 | Documentation | B | README accurate, CHANGES current; tutorial needs rewrite (#438) |
| 11 | Error Handling | B+ | Errors returned not panicked; one defensive panic remains in fork tracking |
| 12 | Extensibility | B | Handler chain extensible, config validators composable |
| 13 | Repo Hygiene | B+ | Clean atomic commits, pinned deps, no junk; no CI configured |

**Overall: A-**

**Score history: B- (Apr 25) → B+ (Apr 26 AM) → A- (Apr 26 PM)**

## Top Strengths

- **PipeEndpoints** (endpoint.go:24-56) — bidirectional message relay with independent goroutines per direction, providing natural backpressure through OS pipe buffers with zero application-level buffering
- **Test suite** — 217 tests: unit tests for every production file, integration tests for security/regression/edge cases/backpressure, cross-platform testcmd binary, ephemeral ports
- **Security posture** — symlink boundary check (handler.go:167-183), origin validation with unit tests, environment whitelist isolation, mutual TLS support, gosec clean
- **Decomposed hot path** — ServeHTTP is a clean handler chain; config parsing delegates to 7 testable pure functions

## Remaining Issues

1. **MEDIUM [Error Handling]** `libwebsocketd/http.go:252` — `panic("Cannot deplete number of allowed forks...")` in `noteForkCompleted()`. Defensive check for a should-never-happen condition, but a panic kills the entire process. Should log and continue.

2. **LOW [Clarity]** `libwebsocketd/launcher.go:44` — `return ..., err` where `err` is guaranteed nil. Should be `return ..., nil` for clarity.

3. **LOW [Infra]** No CI configured. Tests run locally but not on push/PR.

## Architecture Assessment

The architecture is now clean across all layers. ServeHTTP delegates to focused handlers (serveWebSocket, serveCGI, serveStatic, serveDevConsole). PipeEndpoints runs each direction in its own goroutine, enabling synchronous Send() with natural backpressure. Config parsing is decomposed into 7 pure validation functions. The Endpoint interface remains the strongest design element — it cleanly decouples WebSocket I/O from process I/O.

The main structural limitation is that the project has no plugin or middleware system — adding a new handler type still requires editing serveHTTP. For websocketd's scope this is appropriate; it's not a framework.

## Documentation vs Reality

- **handler.go:25** — URLInfo comment removed, but URLInfo is still embedded in WebsocketdHandler when only FilePath is used in 2 places. Minor over-exposure, not a doc issue.
- **Tutorial** (#438) — User-reported confusion about step ordering, port configuration, and --staticdir vs --devconsole interaction. Not yet addressed.

## Quick Wins

1. **Replace fork panic with log** — http.go:252, change panic to log.Error + return. Single line.
2. **Fix launcher return** — launcher.go:44, change `err` to `nil`. Trivial.
3. **Add GitHub Actions CI** — run `go test ./...` on push. ~20-line YAML file.
