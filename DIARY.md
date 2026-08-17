# Engineering Diary

Latest entries first. Record significant decisions, architecture changes, and non-obvious context.

---

## 2026-08-17 — Second-model audit follow-ups (XSS, DoS, TLS hardening)

A second model ran a security audit after the CGI/static fixes. Reviewed and
empirically confirmed all six findings (no false positives); working through
them as atomic commits, test-first. Notes on the non-obvious calls:

Reflected XSS in the dev console (#3): serveDevConsole substitutes
`h.TellURL("ws", req.Host, req.RequestURI)` into `value="{{addr}}"` with no
escaping. net/http surfaces a raw `"` in the request target verbatim in
RequestURI, so `GET /"><script>…` broke out of the attribute and executed on
load — confirmed live. Fixed with html.EscapeString, *not* by switching to
req.URL.Path: the console intentionally echoes the query string into the ws
URL, and URL.Path would silently drop it. EscapeString turns `"` into `&#34;`,
which is sufficient inside a double-quoted attribute.

## 2026-08-17 — Readiness that proved the wrong thing

`TestENV008_UniqueID` failed once on the ARM64 runner with `connection reset
by peer` and — the useful clue — *empty* captured stdout and stderr. It passed
on rerun. Empty output was the thing worth chasing: websocketd logs a startup
banner immediately, so a server that had actually run would have left
something behind. Ours had barely started, yet `waitForPort` had already said
it was ready.

The mechanism: `freePort` binds :0, reads the port, closes the listener, and
returns. The port is then only reserved by convention until websocketd binds
it. When something else takes it in that gap, websocketd exits (`bind: address
already in use`, exit 3) — but the readiness check was a bare TCP dial, which
happily connects to whoever *is* holding the port. So the harness returned a
"ready" server that was not running; the test then hit the squatter as it went
away and got an RST, and cleanup killed our still-initializing process before
it had flushed a single line. Every part of the confusing symptom follows from
readiness proving "someone is listening" instead of "our server is listening".

First attempt at a fix was to also watch for the process exiting. The new test
immediately showed why that is not enough: the dial can succeed against the
squatter *before* our process has even reached its bind, so the check returns
ready before there is any exit to observe. Ordering, not detection, was the
problem.

What actually works is proving identity. `waitReady` now sends a GET for a
path unique to that server and waits for it to appear in *our own* captured
access log — only the process we started can put it there. Around that:
`freePort` never reissues a port within the binary, a lost race is retried on
a fresh port, and a server that exits during startup surfaces its own log
(so "address already in use" is now the error message rather than a mystery
reset two calls later).

Tradeoff accepted: the probe leaves a few 404 access-log lines in the capture,
and readiness now depends on ACCESS-level logging. Every current caller uses
`--loglevel=access`; one that didn't would time out with a message naming the
port, which beats the failure mode this replaced. The four tests that build
their own `exec.Cmd` and call `waitForPort` directly still have the weaker
check — they benefit from the port de-duplication but not the identity proof.

Also set `fail-fast: false` on the test matrix. This flake cancelled the
Windows job 57s in, so the run said nothing about the platform being waited
on. A matrix that stops at the first red runner is least informative exactly
when it matters.

---

## 2026-08-17 — The Windows CGI test asserted the wrong thing (not a hole)

`TestResolveCgiPathBackslash` was red on Windows CI from the moment the CGI
confinement landed. It looked like a traversal gap, but reading the failure
output carefully says otherwise: the paths it complained about were
`C:\srv\cgi\windows\system32\cmd.exe` and `C:\srv\cgi\evil.bat` — both
*inside* the CGI directory. Nothing escaped.

The mistaken premise was that a `..\` segment survives a clean performed in
slash space. It doesn't: `filepath.ToSlash` on Windows rewrites every `\` to
`/` *before* `path.Clean` runs, so backslash dot segments fold back inside
CgiDir exactly like `../` does — which is the accepted, deliberate behavior
for the slash cases in the same table. The test demanded refusal for the one
spelling and fold-back for the other.

Fixed the test to assert what actually matters — containment, checked with
`containsPath` on every accepted result, independent of the expected string —
and corrected the comment in `resolveCgiPath` (and the diary entry below)
that described the nonexistent mechanism. The `containsPath` call stays: it
is genuinely not load-bearing today, and is now commented as a fail-closed
guard rather than as the thing catching Windows separators.

Lesson worth keeping: a red security test is not self-evidently a real
finding. This one encoded a wrong belief about the platform and then failed
loudly enough to look like proof of the bug it was wrong about. Read what the
assertion actually printed before believing its name.

Note: the corrected expectations for the two pre-existing cases come straight
from the Windows CI output, so they are confirmed; the three cases I added
are derived from `path.Clean` semantics and still need a Windows CI run to be
observed green.

---

## 2026-08-17 — Default branch renamed master -> main

The rename is silent in most places but not all: GitHub Actions `branches:`
filters are literal, so the Benchmarks workflow (`push`/`pull_request` on
`[master]`) simply stopped firing rather than failing loudly — the kind of
breakage that only shows up as a gap in the benchmark history weeks later.
The Tests workflow was unaffected because it uses bare `on: [push,
pull_request]` with no branch filter, which is the more rename-proof form.

Also fixed: hardcoded `tree/master/` links in README and the per-language
example READMEs (GitHub does not redirect a deleted branch name, so those
404), and the `git push --tags origin master:master` line in the release
runbook. The gh-pages branch that the benchmark action publishes to is
untouched by the rename.

Worth remembering for future workflows: prefer no branch filter, or accept
that any filter is a hardcoded branch name that needs auditing on rename.

---

## 2026-08-17 — Static file server followed symlinks out of --staticdir

Follow-up to the CGI fix below. While auditing the sibling path handlers, the
static server (`http.FileServer(http.Dir(StaticDir))`) turned out to serve the
contents of any symlink placed inside StaticDir that points outside it —
standard Go stdlib behavior. `http.Dir` blocks `..` traversal but does not
resolve symlinks, so a link is followed wherever it leads. Read-only
disclosure, not RCE, and it needs a symlink to already exist inside the
served directory, so lower severity than the CGI hole — but still a boundary
the server should hold.

The two other handlers were already fine: the ScriptDir/WebSocket path runs
`checkPathBoundary` (EvalSymlinks + prefix), and CGI now does too.

Fix: a small `boundedDir` http.FileSystem wrapping `http.Dir`. It defers to
http.Dir for the normal cleaning/`..`-rejection and correct os error
semantics (so missing files stay 404, not 500), then reuses
`checkPathBoundary` on the resolved path and returns os.ErrNotExist if it
escapes. Chose to wrap the FileSystem rather than pre-check in serveStatic so
every Open the FileServer performs (index.html, directory entries) is guarded
uniformly, not just the top-level request path.

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
then `filepath.Rel`-check that the joined path is still contained. (This
entry originally claimed the Rel check was what caught an interior `..\` on
Windows; that was wrong — see the 2026-08-17 entry above.)
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
