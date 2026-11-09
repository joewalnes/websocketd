# websocketd

A small command-line tool (Go) that wraps an existing CLI program and exposes it via WebSocket. Any program that reads STDIN and writes STDOUT becomes a WebSocket server.

## Build

```bash
go build
```

## Test

```bash
go test ./...
```

Unit tests are in `libwebsocketd/`. Integration tests are in `qa/integration/`. `.github/workflows/test.yml`'s `lint` job runs `go vet`, `gofmt -l`, `staticcheck`, and `gosec` on every push and pull request; there is no separate local lint command to run before that.

The `qa/integration` tests build the `websocketd` binary from source in
`TestMain`, but `go test` caches the result on that package's own inputs.
Changing `libwebsocketd/`, `main.go`, or `config.go` does **not** invalidate
the integration cache even though the built binary's behavior changes — a
cached green can hide a real break. After changing any non-`qa/integration`
package, re-run with `go test -count=1 ./...` (and `-race` for concurrency
changes).

## Mistake retrospectives

When you make a mistake (especially forgetting something the user asked for):
1. Acknowledge it directly
2. Identify the root cause — why did this happen?
3. Suggest a concrete project change to prevent recurrence (add a rule to CLAUDE.md, add a pre-commit check, etc.)
Don't just apologize — fix the system.

## Evolving preferences

When the user expresses a coding preference, convention, or correction during a session, offer to encode it into this CLAUDE.md file so it persists across sessions.

## Documentation

Update `README.md` (and any relevant docs) before committing if the change affects:
- Public API, CLI interface, or configuration
- Setup/installation steps
- Feature behavior visible to users

The public website (websocketd.com) lives in `website/` on this branch and
deploys via `.github/workflows/pages.yml`. It is not a separate branch, so
check it alongside `README.md` for the same changes — the tutorial and feature
list there go stale in exactly the same way. `website/CNAME` carries the custom
domain and must not be deleted or renamed.

### Which surface does it go on

The docs site (`docsite/`, served at websocketd.com/docs) is the only surface
allowed to be long; the others are short *because* it exists.

- `--help` is the single source of truth for flag text. `tools/gendocs`
  generates the man page and `reference/cli-flags` from it. Never hand-write
  a third copy.
- `README.md` serves two readers briefly, one evaluating and one about to
  build, and points onward for everything else.
- `website/index.html` is marketing and orientation, with the compact
  10-second counting tutorial approved for the homepage. Keep the language
  examples and launch commands together; flag references, deployment recipes,
  and longer explanations belong in `docsite/`.

`pages.yml` copies `website/` verbatim to the site root, so anything left in
that directory is published. Planning documents go in `docs-archive/`.

## Changelog

Update `CHANGES` before committing if the change affects:
- Dependencies (especially security fixes)
- User-visible behavior, CLI flags, or defaults
- New features or test infrastructure
- Bug fixes referenced by issue number

Latest version goes at the top. Follow the existing format.

Classification: "Breaking change" is reserved for changes that require
operators to act before upgrading (config migrations, removed/renamed flags,
tightened accept criteria). Bug fixes that observably change behavior stay
classified as fixes — fixing a bug isn't breaking a promise. Leading a
version's entry with a "Breaking changes" block is fine; keep it honest.

## License headers

Every Go source file starts with the 4-line BSD copyright header. New files
use the current year ("Copyright 2026 ..."), not the year copied from an
existing file. Existing files keep their original year. When adding a
missing header to an old file, use the year the file first appeared
(`git log --follow --format=%ad --date=short -- <file> | tail -1`) —
don't assume 2013.

## Test-first

Before implementing a feature or fix:
1. Write a test that captures the expected behavior
2. Run it — verify it **fails** (if it passes, the test isn't testing the right thing)
3. Implement until the test passes

## Test precision

When a test captures a process's output, capture stdout and stderr
separately and assert on the specific stream — never merge them. Which
stream something is logged to is part of the behavior under test.

## Pre-commit checks

Before every commit:
1. Run tests: `go test ./...` — do not commit if tests fail
2. If the change affects user-visible behavior, dependencies, or bug fixes: update `CHANGES` in the same commit (not a follow-up)

## Commits

Break work into small atomic commits — one logical change per commit. Don't bundle unrelated changes. A bug fix, a new feature, and a refactor are three commits, not one.

## Engineering diary

Maintain `DIARY.md` — add an entry when making significant changes, architectural decisions, or non-obvious tradeoffs. Latest entries at top. Focus on *why* and *context*, not *what* (that's in the commits).

## Bug tracking

Bugs and tasks are tracked in GitHub Issues. Use `gh issue list` to view and `gh issue create` to add new ones.

## Agent operations

Configuration for `/go-team` and any other unattended multi-agent run.

**Setup version:** project-setup 2026-09-02

**Requests lane:** `ASKS.md`. Joe's own requests live there and outrank
anything an agent invents for itself. One agent must always be working its
top open item.

**Lesson ledger:** `LESSONS.md`. Append only when something actually bit;
prefer sharpening an existing entry over adding a near-duplicate.

**Autonomy:** merge locally only. Agents may merge verified work to local
`main`; **nothing is pushed to origin.** `git push` is deliberately absent
from `.claude/settings.json`. Joe reviews before anything leaves the machine.

**Crew size:** 3.

### Verification recipe

Never merge on an agent's report. Run this yourself, in a throwaway worktree
on the agent's branch — never in the shared checkout:

```bash
go build ./...
go test -count=1 ./...          # -race as well for concurrency changes
```

`go test` alone is not sufficient evidence — see the integration-cache note
under **Test** above; `-count=1` is what makes a green meaningful after a
change to `libwebsocketd/`, `main.go`, or `config.go`.

For anything a user would *notice*, also drive the real binary:

```bash
# pick a free port; never a fixed one (see singletons)
PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
./websocketd --port=$PORT --devconsole ./qa/integration/testcmd echo
curl -sS -D- "http://127.0.0.1:$PORT/" | head -40
```

For dev-console changes the drive must be a **real browser**, not curl.
`qa/browser` is its own Go module (own `go.mod`), the same pattern as
`qa/capture`, so that chromedp never enters the module that builds the
shipped binary. Two ways this bites if you forget it, both checked by hand
before writing this down:
- `go test -count=1 ./qa/browser/...` run from the repo root does NOT
  silently pass — it fails outright with `directory prefix qa/browser does
  not contain main module or its selected dependencies`. Loud, not a
  vacuous green, but also not what you wanted.
- The broad `go test -count=1 ./...` from the repo root is the quieter
  trap: it neither errors nor mentions `qa/browser` at all — it is simply
  never attempted, and the overall command can still report a clean pass
  having run zero browser tests.

Run the suite from inside its own module instead:

```bash
(cd qa/browser && go test -count=1 ./...)
```

This builds a fresh binary and drives the chromedp suite against it.
Exercising the mechanism is not exercising the integration — a console test
that never opens a socket, never sends a frame, and never reads one back has
verified nothing a user cares about.

### A pending delegation is not a result

If an agent delegates a question to a subagent, its findings may be cited
only after that subagent's completion notification has arrived. If a report
is due while the delegate is still running, the report says so and marks the
claim unknown. An agent cannot distinguish "the delegate answered" from "I
know what the delegate would say", and neither can its reader.

Name the instrument for delegated research the way you would for a
measurement. "An agent said so" is not an instrument. This is written down
because a round asserted what six named web servers do, labelled it verified
against primary sources, and had invented all of it while its subagent was
still running; the claims reached a merge commit on main before the worker
retracted them.

### Never commit anything that identifies Joe's machine

Docs get written by running real commands and pasting real output, which
is how the tutorial came to ship Joe's actual computer name inside its
example transcripts. Use `example-host.local`, `/Users/you/` (or
`$HOME`), and `you` as placeholders — see `docsite/STYLE.md` §7 for the
full table.

`TestNoLocalMachineIdentifiersInRepo` (root package) enforces this: it
reads the *live* hostname, home directory and account name of whatever
machine is running the tests and fails if any of them appear in a
committed file, plus flags any other real-looking `/Users/<name>` or
`/home/<name>` path. It keys off the current machine deliberately, so it
protects whoever drafts next without needing a hardcoded blocklist.
Paste output freely while drafting; scrub what the test names before
committing. Do not weaken or skip it to get a branch green.

### Shared singletons

Parallel agents collide on these. Each agent gets its own, or the run
corrupts itself:

- **TCP ports.** Always bind port 0 / allocate a free port. A hardcoded
  8080 in a test or a manual drive will fight every sibling agent.
- **Unix socket paths.** `--unixsocket` targets go in the agent's own
  `t.TempDir()`, never a shared `/tmp/websocketd.sock`.
- **Headless Chrome user-data-dir.** Two chromedp sessions sharing a
  profile directory crash each other; give each run its own.
- **`refs/stash`.** Shared across every worktree of this repo — never
  `git stash`. Use a second worktree or `git diff > file` instead.
- **The built binary path.** `qa/integration` builds into its own temp
  dir; don't add a build that writes `./websocketd` from two agents at once.

### Do not touch

Settled decisions. Reopen only with Joe, and read the DIARY entry first:

- **No global SIGTERM handler for unix-socket cleanup.** Rejected: it
  changes Ctrl+C and exit-code semantics for every user to serve a niche
  flag. Startup recovery is the only cleanup mechanism (DIARY 2026-08-17).
- **No `ReadTimeout`/`WriteTimeout` on the main server.** They would kill
  long-lived WebSocket and streaming-CGI responses. `ReadHeaderTimeout`
  only (DIARY 2026-08-17).
- **`--maxframesize` defaults to 1 MiB; `--maxforks` to 1024.** Both are
  deliberate backstops, not capacity plans.
- **`--origin` port matching is strict**, with `:*` as the explicit
  wildcard (issue #473).
- **`--socketmode` does not change the default socket permissions**
  (issue #474).
- **`SERVER_NAME`/`SERVER_PORT` stay Host-derived.** Documented, not
  "fixed" (issue #475).
- **The default origin policy stays permissive** for now; the stderr
  banner plus `--anyorigin` is the agreed shape until the announced flip.
- **`SECURITY_AUDIT.md` stays deleted.** Do not reinstate it.
- **`website/CNAME`** must not be deleted or renamed.

### Asset locations

Screenshots and recordings of the dev console go in `website/img/console/`
so the site and the docs share one source. Reference them from `README.md`
and `website/index.html` rather than duplicating files.

## Structure

- `main.go` — entry point, flag parsing
- `config.go` — configuration types
- `help.go` — help text
- `version.go` — version info
- `libwebsocketd/` — core library (WebSocket handling, HTTP, process management)
- `examples/` — example scripts in various languages
- `release/` — release/packaging scripts
