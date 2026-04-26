# Codebase Scorecard: websocketd

**Audited**: 2026-04-25 | **Size**: 25 files, 4.8 KLOC (2.4K src + 2.4K test) | **Language**: Go

| # | Category | Grade | Key Finding |
|---|----------|-------|-------------|
| 1 | Architecture | B- | Clean Endpoint interface; http.go is a god object muxing 5 concerns |
| 2 | Code Quality | C+ | Buffer underflow in origin check, race on canonicalHostname, dead CGI code |
| 3 | Consistency | B- | Mixed snake_case/camelCase methods, inconsistent error handling |
| 4 | Security | B | Solid input handling; symlink following in script dir is unprotected |
| 5 | Performance | C | Regex compiled per request on hot path, unbounded goroutines in Send() |
| 6 | DRY | B- | Terminate() timeout pattern 4x, host:port parsing duplicated |
| 7 | Testability | B- | Endpoint interface is clean; ServeHTTP monolith is hard to test in isolation |
| 8 | Test Coverage | B | 108 tests, excellent integration suite; unit tests minimal (7 total) |
| 9 | Type Safety | A- | Go's type system used correctly throughout |
| 10 | Documentation | B | README accurate, help matches code, QA plans thorough |
| 11 | Error Handling | C+ | Send() can't signal failure, launcher leaks pipes on partial failure |
| 12 | Extensibility | B- | Endpoint interface is extensible; adding handler types requires modifying ServeHTTP switch |
| 13 | Repo Hygiene | A- | Clean history, good commits, no junk files |

**Overall: B-**

## Top Strengths

- **Endpoint interface** (endpoint.go) — clean abstraction enabling bidirectional piping between any two I/O sources via `PipeEndpoints()`
- **Integration test suite** (qa/integration/) — 101 tests covering security, regression, edge cases, performance with ephemeral ports and cross-platform testcmd binary
- **Environment isolation** (env.go, config.go) — whitelist-based env filtering with platform-specific defaults, newline stripping on headers, HTTPS variable blocking
- **Process signal escalation** (process_endpoint.go:34-96) — well-designed SIGINT->SIGTERM->SIGKILL cascade with configurable grace period

## Critical Issues

1. **CRITICAL [Bug]** `libwebsocketd/http.go:295` — Buffer underflow in origin checking. `allowed[len(allowed)-3:]` panics if `allowed` is less than 3 characters. Also at line 289, the loop variable is reassigned (`allowed = allowed[pos+3:]`), so line 295's comparison uses the scheme-stripped value, not the original.

2. **CRITICAL [Performance]** `libwebsocketd/http.go:74` — `regexp.MustCompile()` called inside `ServeHTTP()` — compiles a regex on every single HTTP request. Should be a package-level `var`.

3. **CRITICAL [Resource]** `libwebsocketd/process_endpoint.go:107` — `Send()` spawns an unbounded goroutine per message. Under sustained traffic, goroutines accumulate without limit. Needs a single writer goroutine with a channel, or a semaphore.

4. **HIGH [Race]** `libwebsocketd/http.go:185-197` — `canonicalHostname` is a package-level `var` modified without synchronization in `TellURL()`, called concurrently by multiple request goroutines.

5. **HIGH [Race]** `libwebsocketd/logscope.go:42-44` — `LogScope.Associate()` appends to a slice without holding the mutex, while the log function reads the same slice under the mutex.

6. **HIGH [Bug]** `libwebsocketd/http.go:149-159` — `cgienv` is constructed with ParentEnv + SERVER_SOFTWARE but never used. `cgiHandler.Env` is set to a separate single-element slice. CGI scripts don't receive the expected parent environment.

7. **MEDIUM [Bug]** `libwebsocketd/handler.go:157` — `panic()` on unreachable path-parsing code. Should return an error, not crash the server.

## Architecture Assessment

The core design is sound. The `Endpoint` interface + `PipeEndpoints` pattern is elegant — it cleanly decouples WebSocket I/O from process I/O and enables bidirectional message relay without either side knowing about the other. The process lifecycle management (signal escalation, configurable grace periods) is well thought out.

The main structural problem is `http.go:ServeHTTP()` — it's a 100-line method that muxes between WebSocket upgrades, CGI scripts, static files, dev console, and 404s, while also managing fork limits, origin checking, and header injection. This should be decomposed into a router that delegates to extracted handlers.

The second problem is `config.go:parseCommandLine()` — 200 lines of procedural flag parsing, validation, path resolution, and environment setup in a single function. This makes it impossible to unit test individual validation rules.

## Documentation vs Reality

- **CGI handler environment**: http.go:159 passes only `SERVER_SOFTWARE` to CGI scripts — the carefully constructed `cgienv` variable (which includes ParentEnv) is dead code and never passed to the handler.
- **`--header` help text** says "Add custom response header" — now accurate (applies to all responses after fix), but the help text doesn't clarify the difference between `--header`, `--header-ws`, and `--header-http`.
- **Code comment** at websocket_endpoint.go:15-16 says "CONVERT GORILLA — This file should be altered to use gorilla's websocket connection type" — this conversion was completed years ago but the comment remains.

## Quick Wins

1. **Compile regex once** — move `upgradeRe` at http.go:74 to package-level `var`. One-line fix, eliminates per-request regex compilation.
2. **Fix `noteForkCompled` typo** — rename to `noteForkCompleted` at http.go:220.
3. **Remove stale CONVERT GORILLA comment** — websocket_endpoint.go:15-16.
4. **Use `cgienv` in CGI handler** — change http.go:159 from `Env: []string{...}` to `Env: cgienv`. Fixes CGI environment bug.
5. **Protect `canonicalHostname`** — wrap with `sync.Once` at http.go:185-197.

## Technical Debt

- **ServeHTTP decomposition** — extract WSHandler, CGIHandler, StaticHandler, ConsoleHandler from the switch in http.go. Scope: moderate, single file, but touches the hot path.
- **Send() goroutine management** — replace unbounded goroutine-per-send with a single writer goroutine reading from a buffered channel. Scope: process_endpoint.go only, but needs careful testing for ordering and shutdown.
- **Unit test coverage** — core library has 7 unit tests. handler.go, launcher.go, logscope.go, console.go have zero. Integration tests cover behavior well, but isolated unit tests would catch the origin-checking buffer underflow and the cgienv dead code. Scope: cross-cutting, ongoing.
