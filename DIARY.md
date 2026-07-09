# Engineering Diary

Latest entries first. Record significant decisions, architecture changes, and non-obvious context.

---

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
