# Engineering Diary

Latest entries first. Record significant decisions, architecture changes, and non-obvious context.

---

## 2026-08-17 — CGI directory escape (unauthenticated RCE)

serveCGI built the script path as
`path.Join(CgiDir, "."+FromSlash(req.URL.Path))`. `req.URL.Path` is already
percent-decoded and this handler is not behind a ServeMux at the library
layer, so `../` segments arrived verbatim. `path.Join` *collapses* `..`
rather than rejecting it, so a request path could name any file on the host,
which `cgi.Handler` would then execute with the request body on stdin — an
unauthenticated RCE for anyone running `--cgidir`.

Why the top-level binary partly masked it: `main.go` registers the handler
via `http.Handle("/", …)`, and current Go's `ServeMux` lexically cleans the
path and 301-redirects literal `..`, so the raw traversal PoC gets bounced
before reaching us. That is incidental defense — it does not cover the
symlink escape (a link inside CgiDir pointing out executed fine, confirmed
live), and it is fragile to rely on the mux for a security boundary the
library is supposed to own.

Fix (`resolveCgiPath`): normalize the request path ourselves with
`path.Clean("/"+…)` — a rooted clean path can never keep a leading `..`, so
anything trying to climb out folds back to the root and lands inside CgiDir —
then `filepath.Rel`-check that the joined path is still contained (this also
catches an interior `..\` on Windows that slash-space cleaning never sees).
Symlink escapes are handled separately by reusing `checkPathBoundary`
(EvalSymlinks + prefix check), the same guard the ScriptDir path already
uses. Kept the two checks distinct because they defend different things:
lexical containment vs. real-path containment.

Tests: unit table in `http_security_test.go` drives `resolveCgiPath`
directly (leading/interior/trailing dot segments, dup slashes, Windows
backslash, symlink); integration `cgi_test.go` sends *raw* request lines
(Go's client, like curl, rewrites dot segments before sending, which would
hide the bug) and asserts nothing outside `--cgidir` executes.

## 2026-07-09 — Full repo audit; fixed the things the last audit missed

Re-audited everything against the April scorecard and found the drift you'd
expect from a "finished" cleanup: the Benchmarks workflow had been red on
every run since it was added (keyserver flake during k6 install — nobody
noticed because the Tests workflow stayed green), `go test ./...` failed in
any root container (a test asserted the substring "root" never appears in an
env dump), and the release Makefile would have shipped 0.4.x binaries
labeled MIT built by a Go that can't compile the module.

The interesting bug: both endpoint readers could park forever on their
unbuffered output-channel send when the opposite relay direction died first.
Terminate only unblocked *reads* (kill process / close conn) — nothing ever
unblocked a channel *send*, so each broken binary-mode connection stranded a
goroutine holding a 10MB buffer. Fix: a done channel closed in Terminate,
selected against the send, with `defer close(output)` so a still-live
consumer sees EOF instead of hanging (the naive fix without the defer trades
a reader leak for a relay-goroutine leak — the race matters).

Also learned the integration harness captured only stderr while websocketd
logs everything to stdout, which had quietly turned two tests into no-ops.
Worth remembering: a test that can't fail is worse than no test, because it
shows up in the coverage count.

Process note: this diary had one entry while ~35 significant commits landed.
If the rule is too heavy to follow, thin the rule — but the real cost showed
up in this audit: without the diary, the scorecard's claims had nothing
anchoring them and drifted into fiction (84 vs 111 integration tests).

## 2026-04-23 — Project setup for AI-assisted development

Added `CLAUDE.md` with build/test commands and project structure. Build uses standard `go build` / `go test ./...` (not the Makefile's vendored Go 1.11.5). Bugs tracked in GitHub Issues.
