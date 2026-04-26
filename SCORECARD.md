# Codebase Scorecard: websocketd

**Audited**: 2026-04-26 | **Size**: 25 files, 5.4 KLOC (2.4K src + 3.0K test) | **Language**: Go

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | B+ | ServeHTTP decomposed into focused handlers; config extracted into testable functions |
| 2 | Code Quality | B | Origin underflow, nil pointer, CGI env all fixed; Send() goroutine-per-msg remains |
| 3 | Consistency | B | Extracted functions follow uniform error-return pattern; legacy snake_case methods remain |
| 4 | Security | B+ | Gorilla updated, gosec findings addressed, origin checking hardened |
| 5 | Performance | B- | Regex compiled once; Send() still spawns unbounded goroutines under load |
| 6 | DRY | B | Terminate() timeout pattern 4x; most other duplication resolved by extraction |
| 7 | Testability | B+ | 12 extracted pure functions; Endpoint interface enables clean mocking |
| 8 | Test Coverage | A- | 176 tests (76 unit + 100 integration); covers security, regression, edge cases |
| 9 | Type Safety | A- | Go's type system used correctly throughout |
| 10 | Documentation | B+ | README accurate, CHANGES up to date, thread-safety contracts documented |
| 11 | Error Handling | B | Nil pointer fixed, gosec addressed; Send() still can't signal write failure |
| 12 | Extensibility | B | Handler chain is extensible; adding new handlers is straightforward |
| 13 | Repo Hygiene | A- | Clean history, atomic commits, no junk files |

**Overall: B+**

**Previous score (2026-04-25): B-** | **Delta: +1 full grade**

## Changes Since Last Audit

| Category | Before | After | What changed |
|----------|--------|-------|--------------|
| Architecture | B- | B+ | Decomposed ServeHTTP and parseCommandLine |
| Code Quality | C+ | B | Fixed origin underflow, nil pointer, CGI env, panic |
| Security | B | B+ | Gorilla v1.5.3, gosec fixes, origin hardening |
| Performance | C | B- | Regex compiled once (was per-request) |
| Testability | B- | B+ | 12 extracted testable functions |
| Test Coverage | B | A- | 7 → 76 unit tests |
| Error Handling | C+ | B | Nil checks, error returns replace panics |

## Top Strengths

- **Endpoint interface** (endpoint.go) — clean bidirectional piping abstraction unchanged and solid
- **Test suite** — 176 tests: integration tests catch real bugs (binary deadlock, nil pointer, process hang); unit tests validate config parsing and HTTP routing logic in isolation
- **Decomposed ServeHTTP** (http.go:82-100) — clean handler chain with `serveWebSocket`, `serveDevConsole`, `serveCGI`, `serveStatic` each returning bool
- **Extracted config validators** (config.go) — `resolvePort`, `validateSSL`, `buildParentEnv`, `resolveCommand`, `resolveScriptDir`, `validateDir` all return errors, all unit tested

## Remaining Issues

1. **MEDIUM [Resource]** `libwebsocketd/process_endpoint.go:107` — `Send()` spawns one goroutine per message, serialized by mutex. Under sustained traffic goroutines accumulate. Should be a single writer goroutine with a buffered channel. (Deferred — design under consideration.)

2. **LOW [Race]** `libwebsocketd/process_endpoint.go:18` + `handler.go:77` — `closetime` field is modified after construction (`process.closetime += ...`) but read in `Terminate()`. Not mutex-protected. Safe in practice because modification completes before `Terminate()` can be called, but fragile.

3. **LOW [Race]** `qa/integration/helpers_test.go` — `Server.stderr` buffer accessed concurrently (goroutine writing, cleanup reading). Test infrastructure only, not production code.

4. **LOW [Cleanup]** `handler.go:25` — TODO comment noting `*URLInfo` is over-exposed. Used in 3 places; could be simplified.

## Architecture Assessment

The refactoring significantly improved the two weakest areas. ServeHTTP is now a 20-line method that delegates to focused handlers — each can be reasoned about and tested independently. The config parsing follows the same pattern: `parseCommandLine` orchestrates 7 pure validation functions that return errors instead of calling `os.Exit`.

The core Endpoint/PipeEndpoints design remains the strongest part of the architecture. The main structural concern is now the Send() goroutine-per-message pattern — it works but doesn't provide backpressure. A single writer goroutine with a buffered channel would be more idiomatic and resource-bounded.

## Documentation vs Reality

- **SCORECARD.md** itself was stale (listed old findings as current). Now updated.
- **handler.go:25** TODO comment says URLInfo is used in "one single place" — actually used in 3 places (handler.go:51, env.go:58-59). Minor inaccuracy.

## Quick Wins

1. **Replace Send() goroutine-per-message** with single writer goroutine + buffered channel — fixes resource and backpressure issues in one change.
2. **Initialize closetime in constructor** instead of modifying after creation — eliminates the race.
3. **Protect test stderr buffer** with mutex in helpers_test.go — fixes `-race` detector warnings.

## Technical Debt

- **Send() goroutine management** — Current: unbounded goroutines. Target: single writer goroutine with channel. Scope: process_endpoint.go only, but needs careful shutdown/ordering testing.
- **10 open GitHub issues** — Mix of features (#413, #403, #350), docs (#27, #438, #449), and fixable bugs (#456 ping/pong, #453 CGI paths, #448 slow connect, #445 frame size). These represent the next phase of work.
