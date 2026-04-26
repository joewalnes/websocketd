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

Unit tests are in `libwebsocketd/`. Integration tests are in `qa/integration/`. No linter is currently configured.

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

## Changelog

Update `CHANGES` before committing if the change affects:
- Dependencies (especially security fixes)
- User-visible behavior, CLI flags, or defaults
- New features or test infrastructure
- Bug fixes referenced by issue number

Latest version goes at the top. Follow the existing format.

## Test-first

Before implementing a feature or fix:
1. Write a test that captures the expected behavior
2. Run it — verify it **fails** (if it passes, the test isn't testing the right thing)
3. Implement until the test passes

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

## Structure

- `main.go` — entry point, flag parsing
- `config.go` — configuration types
- `help.go` — help text
- `version.go` — version info
- `libwebsocketd/` — core library (WebSocket handling, HTTP, process management)
- `examples/` — example scripts in various languages
- `release/` — release/packaging scripts
