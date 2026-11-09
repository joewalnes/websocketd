# Engineering Diary

Latest entries first. Record significant decisions, architecture changes, and non-obvious context.

---

## 2026-11-08 - Ship the approved homepage with its short tutorial

The reviewed design puts the counting example back on the homepage. That is
an explicit exception to the earlier orientation-only rule: a visitor can see
an ordinary program, its launch command, and the browser client together,
while the detailed documentation remains a separate surface. Compiled
examples launch their binary or runtime directly; only scripting languages
use a shebang.

The production page uses the existing docs font files and a local SVG sprite
for the approved Lucide icons. It does not depend on the design preview's
icon loader or scripting environment. The existing analytics and the Pages
deployment configuration stay in place.

---

## 2026-09-07 - What qa/plans still holds that no test does

162 of the 317 cases are now asserted by a named test or CI job; the other
155 are the directory's actual value and a future round need not re-derive
which. The manual-only remainder is Windows, real browsers, real clients,
packaging, install, and the language examples — plus two flags nothing
anywhere tests, --closems and --reverselookup, whose only written procedure
is PROC-006/007 and CLI-018. The plans are worth keeping for those.

Match cases to tests by subject, not by number. The integration suite's names
look like the plan ids and are offset by one from CORE-003 and SEC-004 on;
a numeric join is wrong for most of both files.

Staleness here is old, not fresh. Of seven corrected cases none came from
today's path-boundary work — that work only added coverage. SEC-023 has
contradicted the code since the day the plans were committed (main.go already
read r.Host), and four DEV cases describe the console 29d7f87 replaced five
days ago. Nobody had read these files in months, which is the argument for
the per-case **Automated** lines: they decay visibly.

---

## 2026-09-07 - --redirport now carries the path, and why that is safe

The literal "/" in redirectLocation was never a decision. The 2016 commit
that added --redirport wrote it inline ("redirect to same hostname as in
request but different port and probably schema"); nothing since documented
root-only as intended, no test asserted it, and the generated flag docs
already claimed only the scheme and port were rewritten. So: a bug.

Fixing it means client bytes reach a Location header carrying a #nosec G710
that asserts this is not an open redirect. That assertion held trivially
while the path was a constant. To keep it holding for a reason rather than
by accident, the path and query are resolved as an RFC 3986 reference with
no authority (url.URL.ResolveReference) instead of being concatenated -- a
reference cannot introduce an authority, so "//evil.com" cannot become
protocol-relative. r.URL.RequestURI() was the tempting alternative and is
wrong: for an opaque target it returns "http:foo", which concatenation
would splice into the middle of the header.

redirectLocation kept its signature so its five cases stayed untouched.

Two notes on the instrument, both of which changed a finding:

The first probe harness used curl, which normalises "//evil.com" and
"%2e%2e" before they leave the process. Every hostile case had to be
re-driven over a raw socket. The proof the raw sender works is the three
request lines that come back 400 -- an HTTP client would never have put
those bytes on the wire.

Then the raw prober reported header injection on "/a%0d%0aX-Injected:%20yes".
It had not found one. The script tested `b"X-Injected" in response_head`, and
that text appears inside the Location *value*, still percent-encoded. Reading
the raw bytes settled it: one Location line, six response headers, no CR or LF
anywhere in the value. The habit worth keeping is that a positive result is a
claim about the instrument first and the subject second -- re-read the prober,
not just its output.
## 2026-09-07 - Comment sweep: three claims libwebsocketd had stopped honouring

Two thousand lines landed in `libwebsocketd` in a day, so the comments were
audited rather than patched: every block in the thirteen non-test files (122
by a mechanical count) and the test files besides. Three claims were wrong,
across four blocks, plus one on the exit-codes page. The pattern in all of
them is the same: the sentence still parses and its conclusion is still
right, but the fact underneath it had moved. `setupPingPong` promised "any
message" resets the ping deadline, which would make a chatty client immortal.
`resolveCgiPath` argued from there being no ServeMux in front of the handler,
corrected at `normalizeURLPath` that morning in 9a9e487 and left standing at
the two older sites. `createEnv` called five CGI variables "not set" twelve
lines under the code that sets them all to empty. None was decidable by
staring; each was settled by driving the binary with a positive control in
the same run, so a fixture that merely failed to start could not read as a
finding.

---

## 2026-09-07 - The boundary tests got a harness; none of them got weaker

Six files pinning today's path-boundary fixes each grew a private copy of the
same harness: the freePort retry loop three times over, the fixture writers
scattered across five files, and the same request/expect loop hand-rolled at
every site. The trees themselves genuinely differ -- they are what is under
test -- so the shared thing is a vocabulary and four assertions, not a tree DSL.

Proving a security-test refactor is a no-op needs more than green. Two
mechanical inventories were taken before and after: the `=== RUN` subtest names
(213, identical) and every (test, URL) the suite requests (640, identical). Both
were validated by deleting a `t.Run` and a traversal target first and watching
the diffs catch what `go test` did not -- it still exited 0.

assertServes now checks a source marker on every response it sees, which is
wider than the call sites asked for: a CGI script's source contains the text its
output prints, so "200 with the right output" never distinguished a script that
ran from one handed over. Four mutations in libwebsocketd confirmed the tests
still name what broke.
## 2026-09-07 - Consolidating CHANGES buys structure, not volume

0.5.0's block was 609 lines over 115 entries, several rounds each narrating
the same afternoon. Regrouped to 60 entries by subject; the two breaking
changes moved from 360 lines down to the top. Lines barely moved, 609 to 604,
and that is the honest ceiling under "lose no fact a user needs": the volume
is distinct claims, so merging buys a shorter scan, not a shorter read.
Cutting further means ruling that root-cause forensics for surfaces no
released version carried (the docs site's CSS, the test harness) do not
belong in a changelog. Joe's call, not one to take while tidying.

Two claims the code overruled: release/Makefile builds darwin_arm64 as well
as amd64, so the "macOS (amd64)" platform list was stale; and "the rest of
this release is backward-compatible" was already false above it, since the
--unixsocket fix tells operators to change restart scripts.

The loss check needed two goes. Comparing the *set* of flags and issue
numbers could not see #476 deleted from one entry, two others still naming
it. Counting occurrences catches it. An instrument that has only ever
reported "nothing lost" has not been tested.

---

## 2026-09-07 - --dir executes dotfiles, and it is the gap #476 closed elsewhere

Issue #476 made `--staticdir` refuse any path segment beginning with `.`.
Measured today: `--dir` and `--cgidir` have no such rule, and both *execute*.
One server, one path, two answers -- `GET /.env` is 404, and the same request
carrying an `Upgrade` header is 101. Driving the real binary: a live
`.git/hooks/pre-commit` under `--dir=.` runs and streams its output to the
client, and a hook that execs a shell is an unauthenticated remote shell as the
operator. Git's `*.sample` hooks ship mode 0755, so this is real rather than
theoretical; they merely print nothing. `--cgidir` is worse in reach -- plain
GET, no upgrade, no origin check.

What the exec paths do *not* do is disclose. `.env`, `.ssh/id_rsa` and
`.git/config` are not executable, so the launch fails and no content reaches the
client. The gap is execution only, which is why it reads as narrow and is not.

Recommendation recorded, not implemented -- http.go and handler.go were under
lease, and the rule belongs in the containment-predicate consolidation so one
predicate serves all three handlers. Breakage is narrower than it looks: the
rule keys off the request path, so a `--dir` root that itself sits under a
dot-directory keeps working, and an in-tree symlink from a dot-free name still
reaches a dot-subdirectory script -- so no opt-out flag is needed.
## 2026-09-07 - One containment predicate, and where "cannot tell" is decided

A reader no longer has to learn six named answers to "is this path inside
this directory?". There is one, `dirRelation` (with its pre-resolved half),
read as a refusal by `checkPathBoundary` and as a route by `cgiMountPrefix`
-- and its three states are now read at the call site rather than behind a
bool.

`insideDir` was that bool. Its doc claimed one policy -- unknown means
outside, so a caller refusing on the negative must treat outside as
permissive -- but its two callers read the negative in opposite
directions: `boundedDir.Open` serves a file when the exec directory will
not resolve, `staticExecExclusions` keeps the exclusion. One sentence
covering two policies is the shape the eight defects kept living in, so
the wrapper is gone and each site compares against `relInside` itself and
says what `relUnknown` means there. The four-way table lives once, on
`dirRel`.

`containsPath` survives, reclassified. It is not a seventh implementation:
its input is a path `resolveCgiPath` built one line earlier, so `Rel` can
never begin with ".." -- zero escapes over 84 hostile (dir, URL)
combinations. Deleting it is also not the no-op it looks like, which is
why it stayed: with `cgiDir == ""` it changes 32 answers from refusal to a
resolved path, unreachable only because `serveCGI` returns first. A
removal whose safety rests on a caller's early return is not a refactor.

Kept deliberately: the prefix fast path in `checkPathBoundary` (an
accelerator, and the test that proves it cannot disagree stops being a
tautology only while it is separate), and `dirRelationResolved` (folding
it into `dirRelation` would pay a second `EvalSymlinks` per request on
the walk and change one error string).

The whole executable change is five hunks, each an inline of an identical
expression. Evidence: a 1215-pair differential sweep, byte-identical, with
the instrument first shown to catch four deliberate inversions -- and
shown NOT to catch two others, which is the more useful half. Plain
`diff(1)` also reported "Binary files differ" on the first sweep, printing
no lines: an instrument that looks exactly like "no differences".

---
## 2026-09-07 - Auditing --help found a flag value nobody had written down

Read all 31 `--help` entries against the code, then drove the binary for
every claim. Seventeen were wrong, stale or short of a fact a reader
needs; fourteen were already right, which is worth recording -- an audit
that lists only its edits does not say what it checked.

The find that justified the round was not in the entries at all:
`--loglevel=none` is accepted (`LevelFromString` maps it above every real
level) and silences the log completely, and no surface said so. Six levels
were listed as if exhaustive, in `help.go` and in the flag's usage string.

Two other things only the binary would tell you. `--redirport` discards the
request path and query -- every client lands on `/` -- which `--help` and
the generator note both implied otherwise. And `--binary`'s stated
difference (input "immediately flushed") is true of text mode too; the real
difference is the newline, added on input and required on output.

## 2026-09-07 - The boundary check keeps its string prefix and gains a second witness

`checkPathBoundary` ended in `strings.HasPrefix` over two `EvalSymlinks`
results. `EvalSymlinks` resolves symlinks; it does not canonicalize what is
left. On a case-folding filesystem a link target spelling `<base>/page` under
`--staticdir=<base>/PAGE` therefore resolved to a path plainly inside the
directory that shared no prefix with it, and 404ed. Same defect class as the
mount prefix a day earlier, one layer down.

Two things shaped the fix. It runs per request and cannot be cached, so the
prefix test stays as a fast path and the `os.SameFile` walk runs only where
the old code would already have refused -- measured flat on the common path
(11.6us -> 11.4us) and +1.9us on the walk. And the two acceptances are ORed,
not ANDed: requiring both would fix nothing, since the bug *is* the prefix
failing. Admitting identity as a second witness takes a narrow risk on
filesystems that recycle inode numbers, bounded to an ancestor of an
already-resolved path and no worse than what the exec exclusion already
relies on.

`dirRelation` now answers with three states instead of a bool. `insideDir`
and the mount prefix use its negative to *permit*; `checkPathBoundary` uses
it to *refuse*. One value for "outside" and "cannot tell" is safe only while
every caller wants the same thing from the ambiguous case, and that stopped
being true here.
## 2026-09-07 - Proving a screenshot tool unchanged when its output never repeats

qa/capture's main() was 388 lines. Splitting it into named phases is easy; the
interesting part was evidence, because the tool has no tests and its artefacts
are committed to website/img/console/.

Two consecutive runs of the *unmodified* tool are not byte-identical -- the
console stamps every frame row HH:MM:SS.mmm, so ~0.36% of pixels move between
runs and the mp4 varies 17-20 KB with screencast jitter. Byte comparison was
therefore off the table before the first line changed.

What worked: build a noise mask from pairs of same-build runs, dilate it, and
compare only outside it. The first attempt (dilate 2) reported a stable 847-px
difference between builds and looked like a regression -- it was the minute
digit, because both baseline runs happened inside one minute and both new runs
inside another. The same instrument also failed to notice a deliberately
perturbed message, which is what exposed it. At dilate 16 the mask covers 3.9%
of the frame, the perturbation shows up as 226 px in a 21x40 box, and every
before/after pair is 0. A comparison you have not seen fail is not a result.
## 2026-09-07 - Docs for the boundary fixes, written from the code

Five path-boundary defects landed today; the docs described the world before
four of them. Rewrote the `--staticdir`, `--dir` and `--cgidir` entries and the
static-handler half of the security model.

Two judgements worth keeping. `cli-flags.md` is generated end to end, so every
reference change went into `tools/gendocs`' `flagNotes` and was regenerated;
editing the page directly would have failed `docsdrift`. And `--staticdir`'s
entry was already a dense paragraph, so the refusals became a list and the
reasoning moved to the explanation page rather than being appended.

Every behavioural sentence was driven against a built binary, which paid for
itself twice: the exec-directory exclusion is dropped when `--staticdir` names
the exec directory or something inside it, so `--dir=. --staticdir=.` still
serves script source; and `--dir` gets no mount prefix, so a nested script is
reachable only at its `--dir`-relative URL. Neither is in a commit message.

The warning that a secret under an ordinary name is still served survives
unedited. It is what stops a reader concluding `--staticdir` is now safe to
point anywhere.

---

## 2026-09-07 - The CGI mount prefix now asks about directories, not spellings

The #453 mount prefix -- where `--cgidir` sits inside `--staticdir` -- was
derived with `filepath.Rel` over the two flag values. That answers whether the
operator spelled them consistently, not whether one is inside the other. A
release symlink named by one flag and not the other, or a case variant on a
case-folding filesystem, made a nested pair look unrelated: no prefix, so
`GET /cgi-bin/hello.sh` reached neither handler and 404ed a working script.
Failing closed, but the operator's configuration was legitimate.

The static handler's exec-directory exclusion (`insideDir`) had always decided
containment by resolving and comparing `os.SameFile`. The bug was two answers
to one question, so both now go through a single `dirRelation`. Its loop bound
moved from string length to path depth: length only approximates "an ancestor
sits at or above dir's depth" and misses exactly when two spellings of one
directory differ in length -- the class of input at issue.

Identity costs ~19us per call against ~0 for the lexical compare, and
`serveCGI` runs ahead of the static handler on every request once `--cgidir`
is set. The prefix depends only on configuration, so it resolves once behind a
`sync.Once`. Consequence worth naming: flipping a deployment symlink under a
running websocketd no longer re-routes live requests. Restart for a new
layout.

Deliberately not fixed: a `--cgidir` merely *reachable* from the static tree
through a symlink is still not inside it. Making that URL work means searching
the static tree for links into the exec directory -- per-request, and adding
routes where today both handlers refuse. Two tests pin it refused.

That leaves three of the four containment predicates in
`libwebsocketd`; the remaining lexical ones (`checkPathBoundary`,
`containsPath`) are next.

---
## 2026-09-07 - The repo guards scan the index, not the disk

Four guards in the root package each walked the whole tree with their own
`filepath.WalkDir`. The traversal now lives once in `repo_scan_test.go` and
enumerates `git ls-files`; each guard keeps its own predicate, message and
exemptions, which is the part that should stay four things.

The source change matters more than the deduplication. Every guard's contract
is "nothing *committed* contains X", but all four read the working tree, so
any untracked or ignored file counted. A fleet worker's gitignored `.verdict`
that quoted a guard's own failure output turned the next run red for a reason
unrelated to the change under test. One round only stayed green because it
wrote its verdict after the last test run.

Moving to the index also tightened the dead-links exemptions, which had
matched bare base names: *any* nested `DIARY.md` or `docs-archive` directory
was waved through, anywhere in the tree. They are root-anchored paths now.

The first cut of this made "no git index" an error, on the premise that only
CI runs the suite. Wrong premise. GitHub builds a source tarball for every
release tag, and `go test ./...` from one is a normal step for distro
packagers, Homebrew and anyone vendoring — three guards went red there for a
reason the packager could neither cause nor act on. A tarball is the one state
where these guards have no subject, so they skip and say so. It stays a
separate branch from the floor: no repository is a skip, a repository whose
index reaches almost nothing is still a failure. Collapsing the two is how a
guard goes quiet. The disk-walk fallback was not restored, because it would
put the scratch-file bug back where nobody is watching.

The line ledger is close to flat: the four walks were short, and the shared
version does more than any one of them did. The win is one traversal and two
bugs, not fewer lines.

## 2026-09-07 - The diary has no cold tail; the growth is this week's

Went looking for entries old enough to move into a `DIARY-archive.md`, the way
`TODO-archive.md` already works. There is no cold tail to take, so no split
landed.

August holds nine entries, and eight of them are cited by name or still
explain live code:

| Entry | Load-bearing because |
|---|---|
| Unix socket: refuse to take over a live socket | `CLAUDE.md:231` cites it by date |
| Second-model audit follow-ups | `CLAUDE.md:234` cites it by date; `internal/cliflags/cliflags.go` cites it for the `--maxforks` default |
| Static symlink escape | origin of `boundedDir`; the 2026-09-06 `#476` entry refers back to it |
| CGI directory escape | origin of `resolveCgiPath`; direct ancestor of this week's `#453` |
| Windows CGI test asserted the wrong thing | why the `containsPath` call is commented as a fail-closed guard |
| Readiness that proved the wrong thing | only account of why the integration harness proves identity instead of dialling a port |
| The two 2026-08-28 audit entries | four of the "Do not touch" items (`#472`-`#475`) trace their reasoning here |

The ninth, the `master -> main` rename, is genuinely spent and stays anyway:
it sits above six live entries, so lifting it would trade 21 lines for an
archive whose contents follow no rule a reader can state. A chronological cut
is one thing to learn; a cherry-picked archive is N.

What settles it is a number I had not looked at. Entries by month: 1 in April,
1 in July, 9 in August, **19 in September**. Nineteen of thirty entries, and
1169 of 1521 lines (77%), are from September, most from the last two days. The
two genuinely archivable entries, April and July, are 32 lines, about 2%.

So `DIARY.md` is not an old-history problem, it is a this-week problem: the
fleet lands roughly five rounds a day and each writes 75 to 125 lines about
itself. Archiving April and July takes 2% off the bottom of a file whose bulk
arrived in the last 48 hours, and costs a new root file and a second concept
to pull. The lever that would work is shorter entries, not moved ones.
Recorded so the next consolidation round does not re-derive it.

Two notes. `main` moved under this branch mid-round, so the obvious
`git show main:DIARY.md` heading check reports a heading lost that the branch
never had; compare against the branch point. And the em-dash guard that landed
in flight uses an explicit allowlist of live user docs, so diary files sit
outside it and needed no exemption: worth checking rather than assuming.

Dangling reference found, left alone as it predates this round and belongs to
another's files: `docs-archive/EXTERNAL_RESEARCH.md:76` cites "the `--pingms`
/read-deadline work referenced in DIARY.md"; `pingms` appears zero times in
`DIARY.md`.
## 2026-09-07 — Hunting a fourth path-boundary defect, and what the four share

Three path-boundary defects landed in one day (#476 dotfiles and
listings, #453 CGI-inside-static, the relative-boundary fail-open). That
density says under-tested, not unlucky, so this round went looking for a
fourth rather than waiting for one. Two were found and fixed, plus three
fail-closed inconsistencies left alone. Everything below was driven
against the real binary; nothing here is a reading of the source.

**Finding 1, fail open: `--staticdir` served the source of `--dir`
scripts.** #453's disclosure half — never hand back a file the operator
configured to be executed — was attached to `--cgidir` and nowhere else.
The same layout one flag over (`--staticdir=/PAGE --dir=/PAGE/scripts`)
still returned every script as `application/x-sh`, and it is worse there
than it was for CGI: a plain `GET` carries no `Upgrade` header, so it
never reaches the WebSocket handler at all. There is no spelling of the
URL that runs the script and no spelling that does not disclose it —
every request a browser makes for that file is a download. `--dir` is
also the flag websocketd is built around.

**Finding 2, fail open, but obscure: a directory named `index.html` still
got listed.** #476 refuses a directory handle unless `<dir>/index.html`
opens. `net/http`'s `serveFile` asks a stronger question — opens *and is
not itself a directory* — and falls through to `dirList` otherwise. One
input separates the two predicates, and it lands on the listing side. No
operator will build that tree by accident; it is worth closing anyway,
because the guard was relying on another package's control flow rather
than on its own postcondition, and that is an assumption a Go release can
invalidate without anyone noticing.

**What the four (now five) defects actually share.** Not "path handling is
hard". Every one of them is a place where the question *is this file
inside that directory?* gets asked a second time by different machinery
than asked it the first time:

  - `checkPathBoundary` — resolve both sides, compare as strings.
  - `containsPath` — lexical `filepath.Rel`, no filesystem at all.
  - `insideDir` — resolve, then walk parents comparing `os.SameFile`.
  - `cgiMountPrefix` — lexical `Rel` on absolutised paths.

Four implementations of one predicate, and the defects live in the gaps
between them. The relative-boundary bug was two *frames of reference* for
one comparison. #453 was a URL prefix standing in for a filesystem fact.
Finding 1 is a rule keyed to a flag instead of to the property the rule is
about ("configured for execution"). Finding 2 is one predicate that is
strictly weaker than the predicate it exists to enforce. That is a
structural answer, and it suggests a structural fix — one containment
predicate, identity-based, with the lexical variants either expressed in
terms of it or deleted — which is deliberately **not** what this round
did. Collapsing four boundary checks into one is not a change to slip in
beside two security fixes, and each of the three surviving call sites has
a stated reason for its own machinery. Recorded for whoever picks it up.

**The three fail-closed cases found and left alone.** All are the same
crack: `cgiMountPrefix` is lexical, while the exclusion that backs it up
is identity-based. When they disagree the request 404s instead of running
the script:

  - `--staticdir` given as a symlink, `--cgidir` given by its real path
    (or the reverse): `Rel` sees no containment, so no URL prefix is
    derived, and the identity-based exclusion then refuses to serve the
    file the CGI handler declined.
  - The same when the two spellings differ only in case, on a
    case-insensitive filesystem — which is where the #453 reporter lives.
  - A `--cgidir` reached through a symlink inside the static tree.

Each gives 404 where the operator would expect the script to run. I
checked the fail-open twin explicitly and could not construct one: a
lexically-derived prefix only ever maps a URL back into the CGI directory,
where `resolveCgiPath` and `checkPathBoundary` still apply. Fixing this
means resolving symlinks in `cgiMountPrefix`, which is a routing change,
not a boundary change, and belongs in its own round with its own tests.

**Cases driven and found safe**, so that "nothing else found" means
something. URL spellings against `--staticdir`: `%2e`/`%2E` encoded dots,
`%2f` encoded slashes, `%00`, `.git%2fconfig`, `/./`, `//`, trailing
slash and trailing dot, `..` chains, `.well-known/../.git/config`,
`.WELL-KNOWN` case variants, a symlink out of the root, an NFC/NFD
filename pair. Root spellings: `.`, `./`, `sub`, `./sub`, trailing
slash, `/.`, absolute, and mixed relative/absolute pairs — all now agree,
which is the relative-boundary fix holding. `--staticdir=/` and
`--cgidir=/` both serve nothing; the `/` case fails *closed* for CGI too,
which is the thing worth knowing, since the recorded decision to leave it
alone would be wrong if the same reasoning hid a fail-open. Layouts:
`--cgidir` equal to `--staticdir`, inverse nesting, both exec dirs nested
at once, a symlinked `--cgidir` requested by its real name, a symlink from
the static tree into the CGI tree. None disclosed anything.

**Not covered, and worth naming.** Windows is source-read only — no
machine to drive it on, so every claim about `\` separators and
`filepath.ToSlash` here is unverified. TOCTOU between the `Stat`/`Open`
and the `EvalSymlinks` that follows it is real and unaddressed; it is not
in the spelling family and no fixture here would show it. Hardlinks into
a served tree are indistinguishable from ordinary files to any path-based
check, by construction.

**Two things observed and deliberately not fixed.** A single `COMMAND`
sitting inside `--staticdir` still has its source served — same argument
as finding 1, but it is one file rather than a tree, and the tutorial
already teaches putting the page in a directory of its own; excluding a
named file is a different judgement from excluding a directory. And
`--dir` has no dotfile rule, so a WebSocket upgrade to
`/.git/hooks/pre-commit.sample` in a `--dir=.` deployment runs it. That
is `--dir`'s documented contract ("any file under this directory is
reachable as its own WebSocket endpoint") rather than a hole in a
boundary, and narrowing it changes a published mapping.

---

## 2026-09-07 — DOCS_RESEARCH/: two archived, two deleted, and why the split

The previous round named `DOCS_RESEARCH/` as the next candidate and left
it alone. Four files, 76 KB, raw research output from the docs project:
a scan of every commit, a scan of every closed issue, a snapshot of the
then-current doc surface, and a look at comparable projects. The site
they fed shipped, so all four are spent in the ordinary sense. Sweeping
all four into `docs-archive/` would have been a directory-listing
change, and would have got two of them wrong in opposite directions.

**The line I drew: `docs-archive/` records decisions, git records
states.** A file that says *why we did it that way* cannot be
reconstructed and belongs in the archive. A file that says *what the
repository looked like on Tuesday* is a derived copy of something the
repository already holds, and storing it costs a reader the effort of
working out whether it is still true.

**`current-surface-and-drift.md` — stale and wrong. Deleted.** Its title
is "current-state surface", and the surface moved twice since it was
written, both times in the direction it argues against:

- `--sslca`: it records "silently ignored if `--ssl` absent (no
  validation ties them)". Issue #477 landed the same day this round ran
  and rejects it at startup with exit code 1.
- `--staticdir`: it records "no dotfile/directory-listing exclusion",
  and then instructs the manual to write that up as an operator
  responsibility *"since it's not going to be silently fixed as a doc
  change."* Issue #476 fixed it. Dotfiles and unindexed directories are
  refused with 404.

That second one is the shape this seat exists for. It is not a stale
fact; it is a stale *instruction to document a stale fact*, aimed at
whoever writes the next page. Its Part 2 drift audit has the same
problem in bulk: five of the eight rows marked STILL WRONG or STILL
MISSING are done (man page `--maxforks` 1024, `--maxframesize` and
`--anyorigin` present, CLAUDE.md's linter claim, `examples/java`'s
README). Of the two that stand, one is the homepage rebuild in flight
and one is the untagged 0.5.0, which `ASKS.md` already tracks.

A banner would not have saved it, which is the difference from
`MANUAL_TOOLING.md` last round. That document's value was its reasoning,
which survives being wrong about one conclusion. This one's entire value
proposition is that it is authoritative and current, and it is neither;
what is left after the banner is a 14 KB flag table that disagrees with
the shipped reference in two places.

**`commit-inventory.md` — spent, and it is a rendering of `git log`.
Deleted.** Its content shipped: the reverted process pool is
`understanding/design-decisions.md`'s "Why not a process pool", the
dropped BSD/Solaris targets are `reference/platform-support.md`. What is
left is provenance, and I checked whether that provenance was worth
keeping by resolving its hashes here: `efc5ab0`, `e32dd86`, `f7f27db`,
`4dde622`, `aecc484` all name the commits it says they do. `git log -S`
regenerates any row on demand.

**`issues-inventory.md` and `external-research.md` — spent, archived.**
Same spentness, different provenance, and provenance is the whole
argument. These two are distilled from *outside* this checkout: 339
closed GitHub issues, and the contents of the wiki, which is a separate
repository never mirrored here. Neither regenerates from anything local;
`issues-inventory.md` needs network, `gh` auth and a re-read of 339
issues, and upstream issue bodies can be edited or deleted. That is what
`docs-archive/` is for.

I checked the load-bearing claim rather than assuming it. Every issue
number cited anywhere in the live tree — `docsite/content/`,
`tools/gendocs`, the man page, 26 of them — is already carried by
`docs-archive/MANUAL_INVENTORY.md`, which `tools/gendocs:68` cites. The
only two absent from it (#474, #477) are absent from `issues-inventory.md`
too, being newer than the scan. So these files are archived for their own
irreproducibility, not because anything currently reads them.

Their banners name what moved on. The sharpest is `ISSUES_RESEARCH.md`
§0, an "act now, independent of the docs rewrite" list of two items,
both done: the wiki is unlinked from README, website and examples, and
httpoxy (CVE-2016-5385) was already fixed in `libwebsocketd/env.go`
when the file was written. Its sibling drift audit recorded that fix on
the same scan where this file claimed there was none — two research
files, written together, disagreeing about a live security finding, and
the wrong one is the one that sat at the repo root.

**The live-tree leak.** `docsite/PLAN.md:8` told whoever writes a page
that facts come from `DOCS_RESEARCH/` at the repo root — pointing the
next author at the flag table that has `--sslca` and `--staticdir`
backwards. It now names only the archive, and says the code wins where
they disagree. That is the sourcing rule for 44 pages, so it was worth
more than the directory listing. Nothing else had leaked: the shipped
docs are correct on both flags, and `reference/platform-support.md`
gets the dropped BSD targets right.

**One more, found by a check I then decided not to land.** I asked, of
every backticked repo-relative path in every committed file, whether it
resolves. 171 files, four with dangling paths. Three were legitimate:
`DIARY.md` and `docs-archive/` name paths that no longer exist because
that is what a historical record does, and `website/REDESIGN_PLAN.md`
names `website/fonts/` because a plan describes files not written yet.
The fourth was real — `ASKS.md`'s completed A4 entry claims the round
delivered `.github/workflows/docs.yml`. It did not, and should not have:
the docs build is in `pages.yml`, for the same one-custom-domain reason
that reversed the subdomain. Fixed.

**Why that did not become a guard.** Its exemption list would have been
`DIARY.md`, `docs-archive/`, generated `public/` output, and any
document whose job is to describe a future state — and that last
category cannot be identified by a file walker, only by knowing that
`REDESIGN_PLAN.md` is a plan. A guard where the exemptions do the
discriminating is a guard that will be silenced the first time it fires
on a legitimate plan, and after that it scans nothing that matters. The
previous round declined to add a guard for the same reason and was
right; the honest form of this check is a thing you run when you are
about to move files, which is exactly when it fired.

**What I did not do.** I left `DIARY.md:55`'s reference to
`DOCS_RESEARCH/` alone — it is a dated entry describing what was true
when it was written, and rewriting history to keep a path resolving is
how you end up unable to trust the diary. `MANUAL_PLACEMENT.md` stays at
the root; its archival is blocked on A3 landing and finishing it early
would orphan the homepage work. Drift item 4 (the website's 0.4.1
download links) belongs to the A3 seat and item 8 (0.5.0 untagged) is
out of agent reach and already in `ASKS.md`; I recorded both rather than
reaching into another seat's files. Nothing user-visible changed, so
`CHANGES` is untouched.
## 2026-09-07 — A boundary check that was not strict, but meaningless

The report handed to me was "`--staticdir=.` serves nothing", filed as a
usability bug. Half of it was true and the diagnosis was wrong, and the
wrong half is the part worth recording.

`checkPathBoundary` resolved the request path and the configured
directory independently — `EvalSymlinks(path)` versus
`EvalSymlinks(boundary)` — then asked whether the first string had the
second as a prefix. `EvalSymlinks` preserves the relativeness of its
argument. So with a relative boundary the two sides were never in a
common frame of reference, and a comparison between frames does not
merely become stricter. It becomes arbitrary, and arbitrary has two
directions:

- `.` refused everything. Nothing `EvalSymlinks` returns begins with
  `./`, so the prefix could never match and every request 404'd. This is
  the reported bug, and it is the harmless direction.
- `..` allowed everything still spelled with a leading `../` — which is
  precisely what a symlink pointing out of the served tree resolves to.
  `--staticdir=..` served a file outside the root; `--cgidir=..`
  *executed* one. The same tree reached through an absolute path refused
  both.

So the check failed open, and it did so in the configuration a person is
most likely to reach for when the thing they want to serve is one level
up. That reframes the work: not a 404 to tidy up, but a boundary that
silently stopped being a boundary depending on how its argument was
typed. The lesson generalises past this function — a containment
predicate whose two operands can be in different coordinate systems has
no safe failure mode, because "outside" and "inside" are both reachable
by accident.

Worth noting which spellings *worked*, because they are why this
survived: `public`, `./public` and `..` all resolve to plain relative
names that do prefix-match, so the flag looked fine in casual use and in
every existing test, all of which passed absolute paths. Only the
spellings that collapse to `.` were visibly broken, and the one that
fails open was invisible.

**Why the fix goes in `checkPathBoundary` and not in `config.go`.** The
obvious alternative was to absolutise `--staticdir`/`--cgidir` at startup
the way `resolveScriptDir` has always done for `--dir`, which would have
made the symptom disappear without touching the predicate. I did the
opposite, deliberately: the defect is in the predicate, and a caller-side
workaround leaves it armed for the next caller that passes a relative
boundary. Fixing the predicate also fixes `--dir` against a hypothetical
future relative value for free.

`filepath.Abs` on both operands before `EvalSymlinks` on both — not the
reverse. Ordering matters and I checked it on the real binary rather than
reasoning about it: resolving to absolute first means any symlink in the
*working directory's own chain* is resolved identically on both sides, so
running the server from a symlinked directory gives the same answers as
running it from the physical one. Resolving symlinks first would leave
each side to absolutise against a cwd they might disagree about.

The trap here is well known and I assumed I would fall into it: "just
call `filepath.Abs`" is also the shape of change that quietly deletes a
traversal defence. The guard against that was writing the escape cases
first and confirming they were red, in particular the `..` fail-open
which passes trivially if you only ever test that inside-paths are
allowed. Every spelling in the new tests carries both directions — the
file that must be served and the symlink that must not — and each
security assertion is gated behind a positive control that fails the test
if the server was not really serving from the intended root, so a
regression to 404 can never masquerade as a regression to safe.

**Left alone deliberately: `--staticdir=/`.** A `/` boundary makes the
separator-suffixed prefix `//`, which nothing matches, so it serves
nothing — before and after this change. It is the same class of bug, but
"fixing" it means newly exposing the entire filesystem, which is not a
change to slip in beside a security fix. Recorded here rather than acted
on.

**Not done: pinning the roots at startup.** `--staticdir`/`--cgidir` are
still interpreted against the live working directory on every request,
where `--dir` is resolved once at startup. Startup resolution is the
better semantics — the served root is a deployment decision, and the
working directory is process-global state that should not be able to move
a running server's root — but websocketd never chdirs itself, so today
the difference is theoretical. Against that, it changes the startup
banner from the operator's spelling to an absolute path, and
`docsite/content/start/tutorial.md` documents a transcript showing the
relative form. That is another seat's file. Not worth breaking a
published transcript for a hazard nothing currently triggers; flagged for
whoever owns the docsite next.
## 2026-09-07 - An instruction that lives only in a transcript binds nobody

The style rule was given twice, in Joe's own words both times: avoid
llm speak like the em dash, and then, later, apply those changes
everywhere. Both times it was obeyed. Both times it stopped at the
pages that happened to be open in that session.

Measured on the tree before this round: `docsite/content/` held 31 em
dashes across 10 of its 45 pages, `docsite/STYLE.md` held 11 of its
own, `README.md` held 4, and `website/index.html`, rebuilt and merged
about an hour after the second telling, shipped 4 more as `&mdash;`
entities. Fifty in total. Every one of them was written after the rule
was given.

That is not disobedience, and treating it as disobedience is what makes
it recur. The rule was never a property of the repository. It was a
property of one conversation, and conversations do not survive the
agent that had them. The next writer opened a clean tree, found nothing
in `STYLE.md` telling them anything about dashes, and wrote the way
anything writes by default. Asking a third time would have produced a
third round of the same, because the third telling would have expired
the same way the first two did.

So the deliverable was never the sweep. The sweep is the cheap part and
it decays. What changes the outcome is that the rule now exists in two
places that outlive a session: `STYLE.md` 2.11 for the reason, and
`TestNoEmDashInLiveDocs` for the mechanism. A writer who has never seen
the conversation now gets told by the build.

**The entity forms are the reason the check needed writing rather than
scripting.** `grep '—' website/index.html` returns nothing and reports
the file clean. It holds four `&mdash;`, one of them in the `<title>`.
A person who reached for the obvious one-liner would have measured the
homepage as already fixed and moved on. The guard checks the character
and all three HTML spellings.

**Where the line falls between mechanical and advisory matters more
than how much is covered.** The em dash is objective and fails the
build. The wider list Joe gestured at ("honest truth", "load bearing",
"etc.") is in `STYLE.md` 2.12 as a judgement a human applies, and is
deliberately not tested. The evidence for that split is in this
repository: `libwebsocketd/http.go` says a check "is not load-bearing
today", which is the precise technical meaning and exactly the right
word. A checker that reddened the build on that sentence would be
switched off inside a week, and a switched-off checker is worse than an
absent one, because it also carries the false claim that the rule is
enforced.

**Grepping the whole repo for the wider list found almost nothing.**
"delve", "crucial", "seamless", "leverage" as a verb, "in today's
world", "it's not just X, it's Y", "the honest truth": zero hits, in
`docsite/content/` and everywhere else. The A5 rewrite that wrote every
page fresh against source already produced clean prose. So 2.12 is
written honestly as a constraint on the next draft rather than as a
description of a defect, and says so. Shipping a banned-words list with
no instance behind it would have been a guess dressed as a measurement,
and the over-broad checker that follows from it is exactly the failure
mode above.

**The link lists were not a new convention, they were an unfinished
one.** 24 of the 31 content dashes were the same shape: a "See also"
bullet separating a link from its gloss. Seven pages did that; the
other 34 already made the link the subject of a sentence. A5 ran six
writers in parallel and the unifying read afterwards caught voice but
not this. Converging the seven cost nothing and made the glosses say
more, because a predicate will not accept "read this before you rewrite
count.sh" the way a dangling fragment will.

One thing worth flagging for whoever picks this up: the guard exempts
blockquotes, which is what lets `STYLE.md` 2.11 print the wrong example
next to the right one. It also means a writer could evade the rule by
quoting. That is the correct trade, since a quotation has to match its
source, but it is an evasion route and it is not accidental.

---
## 2026-09-07 — The two plans A3 spent, and the rule that outlived one of them

Round 1 left `MANUAL_PLACEMENT.md` at the root under a banner saying
"archive this once A3 lands", and A3 landed as `380a043`. Round 2 drew the
line these two were judged against: `docs-archive/` records decisions, git
records states, and archiving is for what the checkout cannot reproduce.
Both files came out archived, which is the answer round 1 predicted for one
of them, but neither for the reason that was written down.

**`MANUAL_PLACEMENT.md`. The document is spent; its rule is not, so the rule
left first.** The surface boundaries in it are what A3 was built against and
what the next writer will need, and a rule that governs future work has no
business living in a spent plan at the root where nothing points at it.
Sixteen lines in `CLAUDE.md`'s Documentation section now carry it: the docs
site is the only surface allowed to be long, `--help` is the single source of
truth for flag text with the man page and `reference/cli-flags` generated from
it, `README.md` serves two readers briefly, and `index.html` is marketing with
no reference content on it. Two hundred lines of rationale reduced to the
part that decides anything. After that, `git grep MANUAL_PLACEMENT` finds it
only inside `docs-archive/` and in dated diary entries, and it followed its
siblings.

Its own banner had gone stale while sitting there, which is the small lesson
worth keeping. It warned that the document "proposes the subdomain twice";
round 1 had already rewritten both body mentions, so by the time anyone read
the warning the only surviving mention of the subdomain was the warning
itself. A status banner is a claim about a file, and it decays exactly like
the file does. The replacement states what is overtaken without restating it,
and names the one thing in there nobody ever acted on and nothing else
tracks: enabling GitHub Discussions.

**`website/REDESIGN_PLAN.md`. Spent, and it was one deploy from being
published.** Its §6 verification procedure has been run, and §0 had already
turned into the trap round 1 found at the root: §0.4 asks Joe to decide
whether to drop the dead Universal Analytics tag `UA-38494812-1`, which the
A3 build removed. Checking §0 item by item rather than assuming was worth it,
because it is not uniformly spent. Two items are still genuinely open, cutting
a 0.5.0 release and pushing to origin, and both are already in `ASKS.md`,
which is where an open item belongs. One is untouched: §0.5 asks whether to
keep, update or drop the footer's two `twitter.com` credits, and the rebuilt
footer kept them unchanged, so the question stands.

The reason it could not simply stay where it was is the thing this round
actually found. `pages.yml` assembles the site with `cp -R website/. _site/`.
Every file in `website/` is served from the root of websocketd.com, and
assembling the tree the way the deploy does put all 52 KB of the spec in the
artifact root. The next push would have published an internal planning
document at `https://websocketd.com/REDESIGN_PLAN.md`, quoting Joe, recording
that the fleet never pushes to origin, and carrying a measured assessment of
three competing projects. Nothing was checking. The lychee job walks the
assembled tree and passes on it happily, because a markdown file with working
links is not a broken link. `TestNoPlanningDocsInPublishedWebsiteDir` closes
it, and carries a floor for the reason round 2 gave the machine-identifier
guard one: with `website/` present but empty the walk finds nothing and would
otherwise report a clean pass.

It is archived rather than deleted, and the split is per section rather than
per file, which is a case round 2 did not have to face. §3 through §6 are
reproducible or executed: §3 derives from `blueprint.css`, §4 and §5 are
better read as `website/index.html` itself, and §6's results are in `ASKS.md`
and in the entry below this one. §1 and §2 are not: they are rendered
measurements of the old homepage and of Caddy, mitmproxy and Syncthing taken
on 2026-09-06 with headless Chrome and curl, against sites that have already
moved on. Cutting a spec into a kept half and a discarded half would have
destroyed it as a record of what was decided, for a saving of maybe 20 KB, so
the file went whole and the banner says which half is which. `CHANGES`
advertises those measurements by name, so its reference was re-pointed rather
than removed.

Not done, and deliberately: `docs-archive/REDESIGN_PLAN.md` §4.8 still lists
`website/REDESIGN_PLAN.md` as a file that "stays as the record". That is what
was planned, it is wrong now, and editing it would falsify the record instead
of dating it. Same principle as the dangling path round 2 left in a dated
entry.

## 2026-09-07 — Judging the three root MANUAL_*.md planning documents, one at a time

Two consolidation rounds on the Go source came out net-zero, which said
the accretion was not in the code. It was three planning documents from
the docs project sitting at the repo root — `MANUAL_INVENTORY.md`,
`MANUAL_PLACEMENT.md`, `MANUAL_TOOLING.md`, 33 KB between them — left
behind when an earlier round moved `MANUAL_PLAN.md` into `docs-archive/`.
The temptation was to sweep all three after it. They turned out to have
three different answers, and sweeping would have got two of them wrong.

**`MANUAL_TOOLING.md` — spent, and its lead recommendation is actively
wrong.** Hugo, `tools/gendocs` with a drift check, Mermaid shortcodes,
`llms.txt`/`llms-full.txt`, the lychee link check: all shipped as
written. The site *location* shipped inverted. The document recommends
`docs.websocketd.com` and puts `websocketd.com/docs` in a table row
marked "Rejected"; the project shipped the path, because GitHub Pages
allows one custom-domain deployment per repository. Its "action only you
can take" — add a DNS CNAME — is therefore an instruction to do
something the project decided against, and the separate
`.github/workflows/docs.yml` it asks for does not exist and should not.
Archived under a banner naming exactly that.

The interesting part was the blast radius. Two comments in the live tree
had already misattributed the *reversal* to this document —
`pages.yml` and `render-link.html` both cited `MANUAL_TOOLING.md` as the
source of the one-domain constraint that in fact overruled it. Whoever
wrote them reached for the nearest authoritative-looking root file
without reading what it said. That is the specific harm of leaving
superseded advice at the root: it does not just fail to help, it gets
cited for the opposite of its content.

And it had cost something real. Five `examples/*/README.md` files link
their install step to `docs.websocketd.com/start/install/` — dead on
arrival, while `CHANGES` claims every example README points at the new
site. The lychee job runs `--offline` over the built Hugo output, so it
never reads a repository markdown file; the gap was structural, not an
oversight. `TestNoDeadDocsSubdomainLinks` now covers it, matching
`//docs.websocketd.com` rather than the bare host so that *naming* the
rejected subdomain in prose (pages.yml explains why it was rejected)
stays legal.

**`MANUAL_INVENTORY.md` — spent, kept for provenance.** It was a content
checklist, and the checklist has been worked through: `docsite/` shipped
44 pages, A5 re-verified every fact against source rather than copying
it from here, and §6's documentation-debt punch list is done bar the
0.5.0 release, which `ASKS.md` already tracks. What it still holds that
nothing else does is the issue numbers behind each flag note, which
`tools/gendocs` cites so a reader can find the original report. That is
provenance and belongs in `docs-archive/` next to `DOCS_RESEARCH/`'s
role — not at the root, where §6 in particular reads as a live work
queue describing a repo state that no longer exists.

**`MANUAL_PLACEMENT.md` — still load-bearing. It stays.** Its surface
boundaries (homepage points onward, docs site is the only long surface,
README serves two readers briefly) are what `ASKS.md` A3 and
`website/REDESIGN_PLAN.md` are being built against *right now* — five
citations in a plan written hours ago, plus two in the top open ask.
Archiving it would have orphaned live work to make a directory listing
look tidier, which is not consolidation. But three of its facts had
moved on with no way for a reader to tell which parts were still current:
it proposes the `docs.websocketd.com` subdomain twice, it recommends
retiring the wiki (already executed in A4 — README, website and examples
unlinked), and its "item 5 mockups" question was settled by
`docsite/themes/blueprint/`. A status banner names all three, and the two
body references to the subdomain now name the real location. The right
time to archive it is when A3 lands, and the banner says so.

The general rule this round earned: the disposition of a planning
document is not a property of its age or its directory, it is a property
of who still cites it. `git grep` for the filename before deciding,
because the citations are the evidence. Two of these three had none that
mattered; the third had seven, all from work in flight.

Also fixed on the way past: `docs-archive/MANUAL_PLAN.md`'s references to
its siblings had been broken since it was archived without them.
## 2026-09-07 — The homepage shares the docs theme rather than copying it (A3)

The acceptance clause that decided the design of this rebuild was "it
shares type, palette and spacing with the docs site rather than merely
resembling it". Two obvious readings, and only one of them survives a
year.

Copying blueprint.css's `:root` block into `home.css` would satisfy any
screenshot comparison on the day it lands and then start drifting the
first time the docs theme changes a token. The repo already has a live
example of that failure mode: `docsite/static/img/console/` holds
byte-identical copies of the three assets in `website/img/console/`,
duplicated once and now needing to be kept in step by hand. So `home.css`
declares no colour, no font family and no token block of its own, and
`index.html` links `/docs/css/blueprint.css` and `/docs/fonts/fonts.css`
by root-absolute path. `pages.yml` already publishes `website/` at the
artifact root and `docsite/public/` at `/docs` in one deploy, so those
paths resolve on the live domain with no new build step, no new Hugo
mount and no moved file.

The interesting part is the two type roles the homepage needs and the
docs site does not: a 48px display headline and a 20px lede. The
temptation is to type their padding by hand, and that is precisely where
"shares" collapses back into "resembles" — a hand-typed 28.96px is a
copy of an answer, not a use of the formula. blueprint.css applies

    padding-top: calc(var(--lh) - var(--cap) * var(--f))

to the `h1` and `p` selectors themselves, so a more specific homepage
rule that overrides *only* `--f` and `--lh` re-runs the docs' own
arithmetic with new inputs. Measured in the browser: headline
padding-top 28.96px (= 64 − 0.730×48), lede 18.04px (= 32 − 0.698×20),
both boxes whole multiples of the 8px unit, `text-box-trim` computed to
`trim-both` so the modern branch is the one exercised. That measurement
is the acceptance test for this clause — if a number is ever re-typed
here instead of derived, those two values stop agreeing with the
formula, and that is visible in one devtools read.

The cost, stated plainly because it is real: the homepage now cannot be
verified by opening `index.html` from disk or by serving `website/`
alone. Both 404 the stylesheets and the page renders unstyled. Anyone
checking this page has to assemble `_site` the way the workflow does
first. It also means a docs-theme change silently restyles the homepage
— which is the intent, but it needs to be *visible*, which is why the
same round widened the Pages link check to cover the homepage and to run
on `website/**` pull requests. Before that, a homepage-only change got
no CI at all.

Three smaller decisions worth recording:

**Reusing `.site-header` / `.site-header-inner` / `.site-logo`, but not
`.site-footer`.** The header is the same furniture on both surfaces and
its 1200px max-width plus 3U side padding is what the homepage's own
content column is aligned to, so reuse costs nothing and buys identical
chrome. `.site-footer` pads 3U/4U straight off the viewport edge with no
max-width, which would put the footer text on a different vertical line
from the header and the hero — so the footer is `home-footer`, built
from the same tokens at this page's measure. Everything else the
homepage lays out for itself is `home-`-prefixed, because blueprint.css
already owns `.layout`, `.site-nav`, `.content`, `.hero`, `.cta` and
`.section-list` and a collision would style this page as a docs page.

**A still in the hero, the video further down, no third copy.** The old
page stacked `console-light.png`, `console-dark.png` and
`console-demo.mp4` full-width in sequence: 2,217px, two and a half
screens, showing the same feature three times. All three assets are
still used, each once and each doing a different job — the two stills
are a `<picture>` in the hero that matches the reader's own colour
scheme (which is itself the demonstration that the console has both
themes), and the video is the demonstration further down. The video
carries `poster="…console-light.png"`, which fixes a defect the old page
had: with no poster it painted as a white rectangle with browser chrome
until its first frame decoded.

**No hamburger.** Three nav links fit on one row at 390px in the 12px
sans role with room to spare — measured, and the phone captures show it.
The docs site needs a drawer because it carries 24 nav entries; a drawer
for three links is machinery for its own sake.

One instrument note, inherited from the research round and confirmed
here by not repeating its mistake: `Google Chrome --headless
--screenshot --window-size=390,844` does not reliably establish the
layout viewport before first paint and misreported this exact page as
overflowing. Everything measured in this round went through chromedp
with `Emulation.setDeviceMetricsOverride`, an explicitly emulated
`prefers-color-scheme` (this machine is in dark mode, so an unemulated
capture states nothing), and a per-run `--user-data-dir`.
## 2026-09-07 — CGI inside the static tree, and why it was two fixes (#453)

Reproduced against the real binary before touching code: `--staticdir=/PAGE
--cgidir=/PAGE/cgi-bin`, then `GET /cgi-bin/hello.sh`, which came back
`200 Content-Type: application/x-sh` with the script's source in the body.
The control (`GET /hello.sh`, the documented mapping) executed correctly,
so the bug was specifically the handoff between the two handlers, not CGI
being broken.

The reason to treat this as more than a routing annoyance: the file that
came back was one the operator had explicitly configured to be *executed*.
Whatever a CGI script holds — a database password, an internal hostname, a
query — the static handler was handing out on request. That reframing is
what made me split the change in two, because the two halves fail in
different directions and a single rule would have hidden one of them:

  - **Routing** (the feature the reporter asked for). `--cgidir` has always
    mapped the URL path straight into the CGI directory. When the directory
    is nested inside the static root, the URL a browser forms carries the
    directory's own name, and the mapping doubles it. Joe's issue comment
    floated "make --cgidir imply a URL prefix, like Apache's ScriptAlias".
    Deriving that prefix from where the operator already put the directory
    (`filepath.Rel(staticDir, cgiDir)`) gets the Apache behaviour with no
    new flag and no second spelling to teach, and it is inert whenever the
    directories are unrelated — which is every existing deployment. The
    direct mapping is still tried first, so nothing that worked stops
    working.

  - **Disclosure** (the part nobody asked for and the part I would keep if
    only one could ship). The static handler now refuses any file that
    lives in the CGI directory. This is deliberately *not* expressed as
    "URLs under the derived prefix": that would only cover the spelling the
    routing rule already handles. A symlink from elsewhere in the static
    tree into the CGI directory reaches the same file by a URL the prefix
    never sees, and on a case-insensitive filesystem — which is where the
    reporter lives — `/CGI-BIN/hello.sh` is the same file under a name the
    prefix does not match either. So membership is decided by identity:
    resolve the symlinks, then walk the parent chain comparing with
    `os.SameFile`. String comparison would have been cheaper and wrong on
    both counts.

The tradeoff I accepted knowingly: `insideDir` costs a handful of `Stat`
calls per static request, bounded by the resolved CGI directory's own
depth. Cheaper designs existed (cache the exclusion on `WebsocketdServer`,
compare strings), and I rejected the cache specifically because two tests
in this package build a `WebsocketdServer` literal and never call
`NewWebsocketdServer` — a cached field would be silently unset there, and a
security guard that is off in exactly the construction path tests use is
worse than no guard, because it reads as covered. Deriving it from `Config`
on every request cannot go stale.

One configuration is deliberately exempt: a `--staticdir` at or inside
`--cgidir` (`--cgidir=/PAGE --staticdir=/PAGE/htm`). Every static file is
then inside the CGI directory, so the exclusion would leave the static
handler with literally nothing to serve. That layout keeps its existing
behaviour, and there is a test that fails if the exemption is removed.

Every new guard was mutation-tested rather than assumed: matching the
prefix on the raw path instead of the normalized one, dropping the
segment-boundary clause that keeps `/cgi-bindings/` out of the CGI handler,
dropping the static exclusion, dropping the `.`/`..` guards in the prefix
derivation, and dropping the inverse-nesting exemption each produce a named
test failure. The one test that no mutation could make fail is the
traversal suite spelled through the new prefix — normalizing before
matching means those paths never reach the stripping code at all. It stays
in as defence in depth, with a control assertion so it cannot pass
vacuously if the prefix ever stops working.

Not done, and deliberately: the second request buried at the bottom of the
issue — select CGI by file extension or shebang. That turns every file an
application can write into a static tree into a candidate for execution,
which is a much larger change with a much worse failure mode than the one
fixed here. It belongs in its own issue.

---

## 2026-09-07 — --sslca now requires --ssl at startup (#477)

Reproduced first, against the real binary: `--sslca=<file>` with no
`--ssl` started successfully and served plain HTTP — `curl -k
https://...` against it failed the TLS handshake outright, confirming no
listener was even attempting TLS, let alone verifying a client
certificate. The server also never exits on its own in that mode, which
is a second symptom worth naming: reverting the fix and rerunning the new
integration test (`TestIssue477_SslcaWithoutSslRejected`) didn't just
fail an assertion, it hung until the test runner's own timeout killed it,
because `runWebsocketd` waits for the process to exit and the buggy
server just keeps serving.

Root cause was exactly the shape `config.go`'s other SSL validation
already has: `validateSSL(ssl, certFile, keyFile)` checks `--ssl` against
`--sslcert`/`--sslkey` but was never given `SslCaFile`, so `--sslca` sailed
through unchecked. Fixed by extending that function to a fourth
parameter rather than adding a second, parallel validation path —
`--sslca` is exactly the kind of SSL-related flag that function already
owns. The call site (`parseCommandLine`) now passes `*fv.SSLCA` through,
same error shape as its siblings: message on stderr, exit code 1, naming
both flags involved.

Classified as a breaking change, not a fix, following CLAUDE.md's own
rule: "tightened accept criteria" is named there as the example, and
that's precisely what this is — a command line that started today exits
1 tomorrow. The CHANGES entry says what to do about it (add `--ssl` with
`--sslcert`/`--sslkey`, or drop `--sslca`) rather than just what changed.

Confirmed the legitimate mutual-TLS combination (`--ssl --sslcert
--sslkey --sslca`) is untouched: `TestIssue413_MutualTLS` and
`TestIssue413_MutualTLSRejectsNoClientCert` (already in the suite) still
pass, and a new `TestIssue477_SslcaWithSslCertKeyStillWorks` starts that
exact combination and confirms the listener actually binds rather than
merely checking validation in isolation.

---

## 2026-09-06 — /.well-known/ exempted from the #476 dotfile rule

Independent verification of the #476 fix (a second agent driving both
binaries side by side) found one case the blanket dotfile rule was never
meant to hit: `/.well-known/acme-challenge/<token>`, the path ACME's
HTTP-01 challenge requires a domain's own web server to answer
unauthenticated, on plain HTTP. `.well-known` isn't an accidental leak
like `.git` or `.env` — [RFC 8615](https://www.rfc-editor.org/rfc/rfc8615)
IANA-registers it as *the* standard location for URIs meant to be public
(`security.txt`, `apple-app-site-association`, and others live there
too). The blanket rule, applied without thinking about this directory,
would have silently broken a standardized protocol with no flag to
recover it — the same class of silent failure #477 was filed for on the
mTLS side.

So `hasDotSegment` now exempts exactly one thing: a first path segment
that is precisely `.well-known`. Not a prefix, not a pattern — a
directory named `.well-known-evil`, or a dotfile nested *inside*
`.well-known` (`/.well-known/.git/config`), is still refused, and the
symlink-boundary check is untouched and still applies inside the
exemption. The directory-listing rule is also untouched: `/.well-known/`
itself still needs an `index.html` to avoid a 404, same as any other
directory — only the dotfile check gained the exception, since that's
the only rule the ACME/RFC 8615 use case actually needs relief from.

The alternative — leaving the blanket rule and telling ACME users to run
a second web server in front for challenge responses — was available and
isn't unreasonable, but it fails the same test the rest of #476 was held
to: a default that breaks a documented, standard use case with no escape
hatch is a worse default than a two-line, narrowly-scoped exception. If
this exemption turns out to be exploitable in some way not yet
considered, narrowing it further is cheap; broadening the blanket rule to
cover it after the fact (a flag, most likely) would not have been.

---

## 2026-09-06 — --staticdir stopped serving dotfiles and listing directories (#476)

Follow-up to the 2026-08-17 symlink-escape fix on the same `boundedDir`.
That fix closed the boundary-escape hole but left two others open:
`http.FileServer` will happily serve `.git/config` or `.env` if they sit
inside `--staticdir`, and it auto-lists any directory with no
`index.html`. Both were reproduced against a real binary before touching
code (see the fix commit body for the exact curls), then fixed the same
way the symlink check was: inside `boundedDir.Open`, the one place every
static request passes through, not in `serveStatic` or any one handler.

Two decisions worth recording:

**404, not 403, for both refusals.** A 403 tells a prober "something is
here, but you may not have it" — it turns `--staticdir` into an oracle
for the names of files an attacker doesn't already know about. A 404
gives the same answer whether the path is blocked or was never there.
The existing symlink-escape check already made this call (it returns
`os.ErrNotExist`); the two new checks just match it, so all three
boundedDir refusals now look identical from the outside.

**No flag to keep the old behavior.** Someone could plausibly want
directory listings, or a dotfile served on purpose. But `--staticdir`
serving `.git/config` by default was never a chosen behavior — nothing in
the code or the docs decided it, it was just what `http.FileServer` does
when nothing tells it otherwise (see the CLAUDE.md note this issue
prompted about wrong-by-default behavior, not a feature under debate). A
flag would grow the surface to make a security default optional, and
whoever actually wants a listing today can run any other static file
server in front of a directory that also happens to serve WebSockets.
If that need shows up for real, it's a separate, positively-justified
feature request, not a revert of this one.

The `index.html`-presence check reuses `boundedDir.Open` recursively
(checking for `<dir>/index.html` by calling `d.Open`, not the underlying
`http.Dir` directly) rather than a plain `os.Stat`, specifically so a
directory whose only `index.html` is itself a symlink escaping the root
doesn't get treated as "has a valid index" and let through — it goes
through the same dotfile and boundary checks as everything else.
## 2026-09-06 - Researching the homepage before rebuilding it, and one measurement that changed the plan

A3 asks for the websocketd.com homepage to be rebuilt in the docs site's
visual language. The last time a surface here was rebuilt without research
first, four pages were rewritten and the verdict was that the text was
gibberish; the fix turned out to be a framework an hour of reading would
have found. So this round produced a spec and touched no page:
`website/REDESIGN_PLAN.md`.

Three things the measuring turned up that reading the source would not
have.

**The obvious screenshot recipe lies.** `Google Chrome --headless
--screenshot --window-size=390,844` rendered the current homepage as
badly overflowing its viewport: wordmark clipped, nav clipped, hero
running off the right edge. It was the instrument. Old headless does not
reliably establish the layout viewport before first paint. Driven through
chromedp with `Emulation.setDeviceMetricsOverride`, the same page reports
`scrollWidth` of exactly 390 and overflows nothing. A conclusion was one
screenshot away from being backwards, and the spec now carries the
correct procedure as a binding one so the build round cannot repeat it.

**"Shares type, palette and spacing rather than merely resembling" has a
mechanism, and it is better than copying tokens.** `pages.yml` assembles
one artifact: `website/` at the root, Hugo output at `/docs`. So the
homepage can link `/docs/css/blueprint.css` and `/docs/fonts/fonts.css`
directly - verified 200 against a locally assembled `_site`, and
`blueprint.css` contains zero `url()` so it carries no path-prefix
exposure of its own. The better part is the baseline grid. blueprint.css
applies its padding `calc()` to `h1` in terms of the custom properties
`--f`, `--lh` and `--cap`, so a more specific homepage rule that
redefines `--f` and `--lh` on the same element re-runs the docs site's
own arithmetic instead of restating it. Measured on a probe page: a hero
`h1` at `--f: 48px; --lh: 8U` computes `padding-top: 28.96px`
(= 64 - 0.730 x 48) and renders exactly 64.00px tall. The homepage
inherits the machinery, not a copy of the numbers.

**The homepage's most important link is broken, and not by a typo.**
`https://websocketd.com/docs/` returns 404 today, and the live homepage
is a revision from 2026-08-20 that predates the dev-console section. The
docs site A5 built has never been deployed, because the fleet merges
locally and never pushes. Every `/docs/` deep link in the plan is
therefore verified against a locally assembled `_site` rather than
against production, and the spec says so rather than reporting a green.

Also found while measuring, and left alone deliberately: `index.html`
declares no charset at all, so its em-dashes render as mojibake under any
server that does not send one - GitHub Pages happens to, which is the only
reason production looks right. And the docs theme's own `h1` takes 5U at
32px mono where its stated rule would demand 6U; under `text-box-trim`
nothing clips and the site has shipped that way, so the plan gives the
homepage's phone hero 6U rather than copy the discrepancy forward. Not a
bug to fix inside A3.

---

## 2026-09-06 - Rewriting the docs on Diataxis, and what the rewrite found in the code

The first docs site was built fast and reviewed reactively, one screenshot
at a time. Reading it end to end gave a blunt verdict: a jumbled mess.
Three defects, all structural rather than cosmetic. Four pages opened by
describing how the documentation was produced instead of the subject.
Modes were blurred: `guide/` carried conceptual framing next to
per-language buffering fixes, `platform-notes/` was reference material
written as prose, `security/` mixed posture with a list of defaults. And
jargon stacked up undefined, worst in one Windows sentence that packed
three unrelated facts into a single clause.

Patching that would have preserved the shape that caused it. So the site
was specified first and written second: `docsite/STYLE.md` for voice and
the four modes, `docsite/PLAN.md` for the map and the single objective
each page must deliver. Both are binding; a draft that breaks STYLE.md is
wrong by definition, which is a much cheaper review than arguing about
taste.

Six workers wrote 44 pages in parallel on an integration branch, each
owning a disjoint set of files so nobody merged over anybody. That is
also why the last step is a single reader going through every page in
order: parallel authorship buys throughput and costs coherence, and the
complaint that started this was partly that the site read like several
people wrote it.

**The rewrite's real value turned out to be in the code, not the prose.**
Writing documentation against source rather than against the previous
documentation surfaced seven defects that no test covered:

- Two CI jobs were red and had been for some time. `docsdrift` failed
  because the generated flag page had been hand-edited without touching
  the generator. `docs-link-check` could never have passed: it resolved
  `/docs/...` links against `docsite/public`, which holds those pages at
  its own root, giving 1734 errors on an unmodified tree.
- `llms-full.txt` had never been in site-map order, for the same reason:
  a prefix test against `/start/` when `.RelPermalink` reads
  `/docs/start/`.
- Section indexes sorted alphabetically and showed no descriptions,
  reading `.Params.summary`, a field no page has ever set.
- Three statements in `--help` were false: `--closems` claimed signals
  are never sent at the default, `--passenv` claimed it does not work on
  Windows, and `--devconsole` named one of the two flags it is rejected
  with.
- `--sslca` with no `--ssl` silently serves plain HTTP with no client
  certificate verification (issue #477), and `validateSSL`'s error
  message claims to cover `--ssl*` while never being passed the CA file.

That is the argument for writing docs from source. Every one of those was
found by someone trying to state a fact precisely enough to publish it.

The `relURL` trap deserves its own line, because this repo has now hit it
in four separate places: nav, the link render hook, the link checker, and
llms-full. Hugo's `relURL` does not prepend baseURL's path segment to an
input that already starts with `/`. Anything comparing or emitting a
site-root path under a baseURL with a path component has to strip the
leading slash first. A local build looks correct either way, which is
what makes it survive review.

Two areas remain unverified by anyone, and the pages say so rather than
implying otherwise: TLS and mutual TLS were read from source with no real
handshake driven, and every Windows claim is source-read with no Windows
machine available.

---

## 2026-09-06 - The docs theme had never been looked at in a browser

The "Engineering Blueprint" theme shipped with a 90-line comment block
proving its baseline-grid algebra from measured font metrics. The algebra
is correct. The stylesheet was not applying at all.

Line 86 of `blueprint.css` described the derived custom properties as
`--pt-*/--pb-*`. CSS block comments do not nest and have no escape
character: the `*/` inside that glob closed the comment 3 lines early.
The parser then read the rest of the sentence as the start of a selector,
swallowed everything up to the next `{`, and dropped the rule that `{`
introduced -- `:root`, which is where `--u`, both font stacks, all six
font-metric constants and the entire light palette live. The deployed
site served unstyled Times with overlapping lines. Dark mode half-worked,
because the dark palette sits in a later `@media` rule the parse error
never reached; that asymmetry is the diagnostic tell.

Nothing in the repo could have caught this. `hugo --minify` does not
parse static CSS. The theme's own comments assert the math is right, and
they are right, and reading them again proves nothing. It took pointing a
real browser at a real HTTP server rooted at `/docs/` and reading back
`getComputedStyle(document.body).fontFamily === "Times"`.

Everything else found in that pass was downstream of the same absence of
a browser: fonts 404'ing under the `/docs` prefix (the leading-slash
`relURL` trap, surviving inside a static stylesheet where no template
rewrites it), 1200px screenshots with no `max-width` making the page
three viewports wide on a phone, tables with the UA default 1px of cell
padding, every mermaid diagram cropped to a 40px sliver because
`text-box-trim: cap` on the containing `<pre>` trims an inline SVG down
to the font's cap height, a 54px header putting the whole document 6px
off its own grid, and a `#fff`-on-`--accent` CTA measuring 2.23:1 in dark
mode.

The lesson worth keeping: for anything whose output is pixels, a
correctness argument in a comment and a passing build are the same kind
of evidence, and neither is the kind you need. The `qa/capture` script
already established the pattern for the dev console -- allocate a port,
serve for real, drive headless Chrome. The docs site needs the same
habit. Screenshots for this pass are not committed (they are artifacts,
not content), but the instrument is a ~120-line CDP driver over a raw
socket; if this recurs it is worth checking into `qa/`.

One residual, measured and left alone: a nested `<li>` on
`/reference/dev-console/` is 6px off, because `text-box-trim: trim-both`
on a list item lands on the first and last line box of its whole subtree,
and the last one is inside the nested `<ul>`. Every fix for it is either
a magic number or moves the text within the row; it affects one list on
one page and nothing above it.

---

## 2026-09-06 — Landing A4: the full docs site, and what a copy-edit pass is actually good for

Built the whole thing this run: a Hugo site under `docsite/` (custom
"Engineering Blueprint" theme -- grid-paper background, JetBrains Mono
headings, IBM Plex Sans body, baseline-grid-corrected vertical rhythm
with a `text-box-trim` path and a manual-padding fallback), every page
in `MANUAL_PLAN.md`'s site map with real content, the `tools/gendocs`
generator single-sourcing the man page and the CLI reference page from
the live flag set, `/llms.txt`/`/llms-full.txt`, CI (build+deploy,
CLI-flags drift check, a lychee link check), the wiki unlinked, and a
final Fable 5.1 pass over all the prose.

Two things worth writing down before they're forgotten:

**A generator only closes the gap it actually reads from.** Round 1's
content worker read `process_endpoint.go`/`websocket_endpoint.go`
directly and correctly wrote that `--binary` changes forwarding
granularity (raw chunks, no newline gating), not just the WebSocket
frame type -- contradicting `docs-archive/MANUAL_INVENTORY.md`'s original, more casual
claim. But `tools/gendocs`'s hand-maintained per-flag note table (built
by a sibling agent working in parallel, from the same inventory doc) still
had the old claim, so the *generated* reference page and the *hand-written*
guide page shipped saying opposite things about the same flag, on the
same site, for one merge cycle. Single-sourcing the flag table doesn't
single-source the prose *about* the flag unless something actually
checks the two against each other -- nothing did, until the Fable pass
read both pages in the same sitting and noticed they disagreed. Filed as
a real bug, not a style note, and fixed by correcting the generator's
note and regenerating rather than hand-patching the output.

**The Fable pass earned its slot by refusing to overstep it.** Briefed
to tighten prose and leave facts alone, it found six places where it
suspected the content was wrong and flagged all six instead of quietly
rewriting any of them -- including the `--binary` contradiction above,
a reversed code comment in a Redis pub/sub example, an overstated
"generated" claim on the reference index page, a de-facto-reversed
"most common cause" claim between two guide pages, and a "the Origin
always matches" claim that was only true because of a specific
`proxy_set_header` directive present in that page's own example. Every
one of the five real issues traced to a genuine gap (a stale note, a
copy-paste-adjacent mistake, an unstated dependency) once actually
checked against source -- none were the copy-edit model inventing a
problem to seem thorough. The lesson generalizes past this one pass:
a model scoped narrowly and told to flag-not-fix on suspicion is a
second, independent read of content a single-pass writer can't give
itself, and it's worth deliberately keeping that scope narrow rather
than letting a "helpful" pass silently correct things it wasn't asked
to verify.

Also: caught and fixed my own process mistake mid-run -- attached a
throwaway "did it all still build" worktree to the `main` branch ref
itself instead of a detached SHA, which would have let a stray commit
there desync `main`'s ref from the shared checkout's working tree.
Caught before anything was committed; see `LESSONS.md`.

Left open, out of repo/agent reach: the DNS CNAME record for
`docs.websocketd.com`; an actual 0.5.0 release, without which
`start/install`'s download links point at GitHub's generic releases
page rather than a specific working download; and a real decision on
how `docs.websocketd.com` and `websocketd.com` coexist under GitHub
Pages' one-deployment-target-per-repo model, which `docs.yml` states as
an open question rather than papering over.


---

## 2026-09-06 — examples/ content, changelog verified working as-is, wiki links retired from README/website

Third A4 workstream, alongside sibling agents landing `cookbook/` and
`reference/`+`platform-notes/`+`security/`. Scope: `docsite/content/examples/*.md`,
verifying the changelog page, refreshing `llms.txt`, and retiring wiki
links per `MANUAL_PLACEMENT.md`'s RETIRE verdict.

**Changelog: no fix needed.** Built the site and grepped
`docsite/public/changelog/index.html` for real `CHANGES` text ("0.5.0",
the devconsole rebuild description) — round 1's module-mount +
`resources.Get` approach already works correctly. Left it alone rather
than "fixing" something that wasn't broken.

**Examples page.** `docs-archive/MANUAL_INVENTORY.md` §10's real-world examples
(codebender.cc, a cable-modem dashboard, IoT sensor dashboards, a
Pi robotics counter, gphoto2 tooling, a browser terminal game) were
pulled from old support issues, not offered as case studies, and naming
them publicly needs permission nobody has asked for. Wrote them into
`examples/_index.md` as an "in the wild" section citing issue numbers
only (#36, #196, #202, #211, #420, #443) with no project/company names.
The substantive content of the page is the repo's own first-party
`examples/` directory (bash through VBScript, ~19 language directories)
— what each of `greeter`/`count`/`dump-env`/`chat`/`send-receive` does
and how to run it, since that's real code this project fully controls
and has every right to document, unlike the third-party mentions.

**llms.txt reality check.** As of this branch, `main` has `start/`,
`guide/`, `faq/`, `reference/cli-flags.md` (generated), and now
`examples/`/`changelog/` as real content; `cookbook/*`, `platform-notes/`
leaves, `security/`, and three of four `reference/` leaves are still
section-landing-page stubs from the original scaffold commit — confirmed
by `git log` showing those files only touched by the scaffold commit,
never by the sibling branches that will eventually fill them in. Rewrote
`llms.txt` to say "done"/"partial"/"pending" per section instead of the
round-1 blanket disclaimer, so an LLM (or a human) fetching it gets an
accurate map right now, not a snapshot of round-1's intentions. This will
need one more pass once all A4 branches actually merge.

**Wiki retirement (`MANUAL_PLACEMENT.md`'s verdict).** No access to the
wiki repo itself and none needed — the ask is only to stop linking to it.
Repointed four links in `README.md`, one in `website/index.html`'s nav,
and one in `examples/java/README.md` (the path-instruction half of that
file was already fixed in round 1; only the URL itself was still
wiki-shaped) to the new `docs.websocketd.com` paths. Left
`examples/nodejs`, `examples/c#`, `examples/f#`, and `examples/lua`
README files with their wiki install links untouched — out of this
branch's assigned scope, flagged for the foreman instead of silently
expanding scope into files another agent might also be touching.

---

## 2026-09-05 — docs.websocketd.com scaffold: Hugo, a from-scratch theme, and one unresolved infra question

Built the framework half of the new docs site (`docsite/`) per `MANUAL_PLAN.md`/`docs-archive/MANUAL_TOOLING.md`: Hugo (0.165.0, installed via `brew`), a custom "Engineering Blueprint" theme at `docsite/themes/blueprint/`, the nav/section skeleton for the full site map, and the `llms.txt`/`llms-full.txt` pair. Content pages themselves (`start/`, `guide/`, `faq/`, etc.) are sibling agents' work — this branch only carries section `_index.md` landing pages, enough for `hugo build` to produce a real, navigable, empty-content site.

**Baseline grid.** The brief specified U=8px and gave exact font metrics (cap-height/ascent/descent as em-fractions) for JetBrains Mono and IBM Plex Sans, plus a `text-box-trim`-based modern path and a manual-padding classic fallback, with explicit license to use judgment on exact mechanics. The modern formula (`padding-top: LH - cap*F`) is dimensionally sound as given. The classic fallback's literal shorthand isn't — `padding-top: shift; padding-bottom: U - shift` only sums to a multiple of U if the line box itself is already 0 height, which it isn't. Fixed by setting `line-height: 1` (a CSS-defined, UA-independent box of exactly `F`) and deriving `padding-bottom: LH - F - shift`, which sums to `LH` for any F/LH/shift — verified algebraically and left as a comment block in `blueprint.css` so the arithmetic is checkable without trusting the comment.

**Fonts.** Self-hosted both families under `docsite/static/fonts/`, fetched live from `fonts.googleapis.com`/`fonts.gstatic.com` in this session (network reachable in this sandbox, unlike a published Artifact's CDN allowlist — this is a real repo, not an Artifact). Kept only the Latin + Latin-Extended subsets (English-only site) — 4 files instead of Google's default ~12 (drops Cyrillic/Greek/Vietnamese). Discovered mid-way that Google serves both families as full variable-font binaries: a naive per-weight fetch (400/600/700) downloaded three byte-identical files per subset, confirmed via checksum. Collapsed to one `@font-face` per subset with `font-weight: 100 900`, letting the browser interpolate — fewer bytes shipped, same visual result.

**llms-full.txt.** A custom Hugo output format (`LLMSFULL`) on the home page, templated in `docsite/layouts/index.llmsfull.txt`, walking `site.Pages` in an explicit site-map-order prefix list rather than relying on `.Weight`/`.Date` (leaf pages don't exist yet to carry front-matter weights). Verified live: added a temp page under `platform-notes/` with a Mermaid shortcode in it, rebuilt, confirmed it appeared in both the section listing and `llms-full.txt`, confirmed the Mermaid shortcode rendered as `<pre class="mermaid">`, then deleted the page and rebuilt again to confirm all three effects reversed. Also had to disable Goldmark's typographer extension — it HTML-entity-encodes curly quotes, and `.Plain` (which `llms-full.txt` uses) doesn't decode entities back out, so "isn't" was landing in the plain-text output as literal `isn&rsquo;t`.

**CHANGES, single-sourced.** `docsite/content/changelog/` renders the repo-root `CHANGES` file directly via a Hugo module mount (`../CHANGES` → `assets/CHANGES.txt`) read at build time with `resources.Get`, rather than a copied/hand-written duplicate — per `docs-archive/MANUAL_TOOLING.md`. Go's `html/template` auto-escaping handles the raw text safely with no manual escaping needed.

**Left unresolved, flagged not solved:** `.github/workflows/docs.yml` mirrors `pages.yml`'s structure as the brief asked, but GitHub Pages is one deployment target per repository — running both workflows against this repo's Pages settings deploys to the *same* Pages site, last write wins, not two independently-live custom domains. `docs-archive/MANUAL_TOOLING.md`'s "own GitHub Pages target" framing doesn't hold under GitHub's actual model as far as I could determine without live infra access to test against. Documented as a comment in the workflow itself; needs a human decision (alternate host for one site, or some other Pages configuration) before this can go live, same as the already-flagged DNS `CNAME` record.

---

## 2026-09-03 — Landing qa/browser: a chromedp bug, not a console bug, ate most of the debugging

Finishing ASKS.md A1: the chromedp behavioural suite for the rebuilt dev
console had been reconciled onto the wrong DOM contract in an earlier
session (it branched before the console merge) and needed redoing against
what actually shipped. Redoing it surfaced a real, reproducible hang: any
`chromedp.Click`/`chromedp.SendKeys` on a CSS selector, issued after the
console's WebSocket connects, blocked for the full test context budget
(confirmed up to 150s - not slow, genuinely stuck) with zero effect.

Chased it down with a binary search across isolated single-purpose test
files rather than staring at the console source, because the first
plausible-looking hypotheses were all wrong: not the console's own keydown
handlers (no loop, no blocking dialog calls - grepped for both), not the
250ms `setInterval` driving the "open Xs" counter (reproduced with an
identical interval on a synthetic page; no effect), not machine load from
this being a shared box running several other concurrent agent sessions
(reproduced deterministically on a freshly cleaned machine with zero other
Chrome processes running). The isolating fact: `chromedp.Focus` (no
selector re-query) and a raw, selector-less `chromedp.KeyEvent` both
resolved in low single-digit milliseconds on the exact same page in the
exact same state - only `Click`/`SendKeys`, which both internally re-query
the selector through chromedp's `NodeVisible` option before acting, hung.

Root cause, confirmed directly: after the console's connect-time DOM churn
(`state()` and `note()` firing across `#status`, `#connect`, `#frames` in
quick succession), the *live* DOM has exactly one element matching `#send`
(`document.querySelectorAll('#send').length` from real in-page JS says 1,
every time) - but chromedp's own internal node cache, queried via
`chromedp.Nodes('#send', ...)`, returns two. `NodeVisible`'s poller waits
for every returned node's box model to resolve; the phantom second node
never will, so it spins forever. This is a chromedp v0.16.0 defect in its
DOM-node-tracking cache, not anything wrong with the console's markup.

Worked around it rather than waiting on an upstream fix: `clickSafe`/
`typeSafe` in `qa/browser/harness_test.go` resolve the node with a plain,
non-visibility-gated `Nodes()` query and act on it directly
(`MouseClickNode`, or `Focus` + a selector-less `KeyEvent`), which never
invokes the buggy poller. Both mirror `Click`/`SendKeys`'s own "act on the
first match" semantics for selector parity. Every post-connect DOM
interaction in the suite goes through one of the two now.

Once the suite could actually run, it caught a second, genuine bug on its
first real pass: repeated Up presses in the send-history recall only
worked once (see the CHANGES entry). That one was in the console, not the
harness, and is fixed the ordinary way - this diary entry is about the
harness bug because *that* was the one that looked, for a long while, like
it might be the console's fault instead.

Also reconciled `libwebsocketd/console_serving_test.go` against the
existing `console_test.go`: one literal name collision
(`TestDevConsoleSecurityHeaders`), resolved by keeping the version that
parses the CSP into directives (stronger - a substring match can pass
against a malformed policy with the right words in the wrong place) and
folding in the three assertions the older version had that the newer one
didn't. Two more near-duplicates resolved by keeping whichever
implementation was actually stronger on inspection, not by preference:
`console_test.go`'s request-independence test (covers a raw quote-breakout
injection and a bare Host that the other version's `net/http` client can't
even construct) beat the newer file's version, so the newer one was
dropped; the newer file's external-resource scanner (position-aware regex
matching only real src=/href=/url()/@import loads, with its own
false-positive/false-negative unit tests) beat `console_test.go`'s
fixed-substring scan, so that one was dropped instead. The size-budget
test the newer file added checks the *served* response rather than the
authored file, and caught something real in the process: the served body
runs ~1.3KB over ASKS.md's informal "~25KB" once the BSD license text is
expanded into it - inherited from the original console, not new here. Set
the budget honestly to reflect that rather than testing the wrong
artifact to keep a number.

---

## 2026-09-02 — A1: the dev console, rebuilt as a split inspector

The old console was a decade-old scrolling log with a text input at the
bottom. It answered "did something come back?" and nothing else: no byte
sizes, no timestamps, no way to look at one frame, no binary handling worth
the name, and an unbounded DOM that grew until the tab died.

The layout is now a split inspector — a dense frame list beside a detail
pane — because the two jobs are different jobs. Scanning a transcript wants
one line per frame; understanding one frame wants the whole payload, its
opcode, its size, and a choice of pretty / raw / hex. Trying to do both in
one column is what made the old one a log rather than a tool.

Aesthetic is "Teletype": monospace everything including the chrome, no
border radii, hard rules, a solid inverted masthead, sent frames as inverted
rows, one red reserved for errors. Direction is legible without reading:
outbound rows are the photographic negative of inbound ones. The site
redesign (A3) follows this, not the other way round.

Three theme states, not two. "Auto" follows the system and is the default;
an explicit light/dark override persists in localStorage and can be returned
to auto. The state lives in `data-theme` on `<html>` and is applied by the
head script before first paint, so there is no flash of the wrong theme.

Notes on things that were not obvious:

- **The page is a constant.** Dropping the `{{addr}}` substitution was worth
  a commit of its own. It was already dead (the field is filled from
  `location.href`), it was the reflected-XSS surface, and removing it lets
  the response carry a strong ETag and a hash-pinned CSP. The CSP hashes are
  computed at init from the embedded bytes; a hand-copied hash would go
  stale on the next edit and the console would silently come up blank.
- **CSP shapes the code.** No inline handlers, no `style` attributes, no
  `setAttribute('style', ...)`. Everything is classes and `data-` attributes;
  the one CSSOM write is `--vh`, which CSP does not govern.
- **Binary frames use `arraybuffer`, not `readAsBinaryString`** — the latter
  is deprecated and mangles anything above 0x7f, which is precisely the data
  you opened a hex view to look at.
- **The transcript is a ring buffer** (500 frames) flushed on
  `requestAnimationFrame`. `websocketd --devconsole yes` used to grow the DOM
  without bound; it now costs a constant number of nodes.
- **320px took a `minmax(0, 1fr)`.** The body grid sized its column to
  content, so the address bar's intrinsic width pushed the page to 395px and
  the whole thing scrolled sideways on a phone. Measured in headless Chrome,
  not guessed: hiding each child in turn found the row that demanded the
  width.
- Received text is only ever written with `textContent`. Point the console
  at someone else's server and every frame is hostile input.

Deliberately not built, per the agreed scope: saved collections, multiple
simultaneous connections, subprotocol/header editors, scripting, export
beyond copy. Each of those is how a console turns into Postman.

---

## 2026-09-01 — A2: warn loudly, change nothing (yet), and make opting out explicit

The CSWSH finding (default accepts any origin) stayed behavior-compatible:
webpage-in-a-browser → spawn-and-drive-the-served-command is exactly the
exposure, but flipping the default to `--sameorigin` breaks legitimate
non-browser automation and every existing quick-start. So the fix is a
prominent stderr banner when no policy is configured — what the exposure is,
the risk of changing nothing, the three options, and a dated announcement
that a future version WILL default to `--sameorigin` — plus `--anyorigin` as
an explicit, validated opt-in that silences it. The flag matters more than
the warning: when the default flips, `--anyorigin` is already the migration
path and already in the wild.

Banner-on-stderr was a correction worth noting: it bypasses the log
machinery by design (security notice, not a log line), so it must not live in
the stdout log stream — daemons run `websocketd > access.log`, and a stdout
banner would be buried exactly where "hard to miss" must hold.

Also closed the remaining A10 INFO items (panic guard, WriteHeader noise,
64KB binary buffer, crypto-random UNIQUE_ID) and recorded WONTFIX rationale
for the rest: stdout line buffering (the operator's own child can exhaust
memory unaided — the pipe adds no power), reverselookup DNS (opt-in, rare;
timebox if it ever matters), directory listings (operator chose the root).

## 2026-08-28 — Closing out the audit's low-severity items (#472–#475)

Follow-up to the audit remediation above, one commit per issue. Two calls
worth recording:

- **#473, strict `--origin` ports.** The old behavior (portless entry matches
  any port) was documented, but implicit-and-dangerous is still dangerous:
  allowlisting a host vouched for every service on its other ports, and an
  origin from any of them satisfied the check. Went strict (default ports
  only) with `:*` as the explicit opt-in, accepting the breakage — the
  migration is one suffix, and the default-port case that every existing
  test exercises is unchanged. The wildcard is stripped *before* `url.Parse`
  because a `:*` port makes the whole entry unparseable — that ordering bug
  was caught by the new unit tests, not by eyeballing.

- **#475, docs over "fix".** Considered deriving `SERVER_NAME`/`SERVER_PORT`
  from the actual bind address, then didn't: Host-derived values are
  virtual-host semantics, identical to `net/http/cgi` and to how every
  mainstream framework builds URLs, and bind-derived values (an interface
  IP) would break scripts that generate links. The real gap was that the
  trust boundary was invisible — so it is now documented at the point of
  use (README + `--help`), including the `PATH` layout disclosure that the
  default `--passenv` implies.

Also: `--socketmode` (#474) deliberately does *not* change the default —
under umask 0 it would have broken legitimate group-shared sockets — and
negative `--maxframesize` (#472) is now a startup error, since it silently
meant "unlimited".

## 2026-08-28 — Second security audit: remediating what a "wrap any command" tool inherits

The second audit pass (SECURITY_AUDIT.md) went after live exploitation instead
of code reading, and four findings got fixed tonight. The two that involved
real design decisions, not just bug fixes:

- **Process-group teardown (A4).** websocketd's contract is "the process is
  the connection", but signals only ever reached the direct child, so a
  script that backgrounded children leaked them past disconnect — and via
  inherited pipes kept the session and its `--maxforks` slot alive after the
  wrapped process had exited. Children now run in their own process group;
  teardown signals the group and finishes with a SIGKILL sweep once the
  direct child is gone. The deliberate tradeoff: the sweep kills stragglers
  without a graceful window, so anything that must survive has to opt out
  with `setsid` — the Unix-standard way to say "not my session's lifetime".
  The other half of A4 (a *live* client holding a session open while a
  grandchild holds the pipes) is intentionally unchanged: closing on
  direct-child exit would cut off the legitimate fork-a-worker-then-exit
  pattern.

- **Stderr chunking (A1).** The stderr pumps died on any read error, and
  `ReadSlice` returns `ErrBufferFull` for >4KB un-newlined writes — a
  one-message remote wedge. The fix chose bounded-memory streaming (partial
  chunks, like the audit's original framing) over `ReadBytes`-style unbounded
  line buffering: stderr must never be the channel through which a child
  OOMs the daemon. Consequence worth knowing: very long stderr lines now
  arrive as multiple tagged messages under `--passstderr`.

A3 (redirect address built by first-colon splitting — fatal for IPv6 +
`--redirport`) and A5 (control-character escaping at the `logfunc` boundary)
were mechanical. A2 (CSWSH: the default origin policy accepts everything,
including `null`) is deliberately unfixed — changing it is a breaking
behavior change and needs a maintainer decision; it stays documented in the
audit. A6–A9 are filed individually as #472–#475 for triage rather than
bundled, so each can be accepted, declined, or deferred on its own merits.

## 2026-08-17 — Unix socket: refuse to take over a live socket

Prompted by #471 (a duplicate feature request for `--unixsocket`, already
shipped for #435). Reviewing it surfaced a real bug and a design temptation;
only the bug got fixed.

The bug: `serveUnixSocket` unlinked any socket file at the path before
binding, without checking whether anything was listening. Start a second
instance on a path a healthy one already owns and the newcomer silently steals
it — the incumbent keeps running and keeps its children, but nothing can ever
reach it again, and neither process logs a thing.

Fixed by probing before unlinking: `net.DialTimeout` on an AF_UNIX path either
connects (someone is listening → refuse to start) or is refused immediately
(stale → safe to remove). The framing that settled the design is TCP parity —
a TCP listener already refuses a port that is in use, and the Unix socket was
the odd one out in taking one over.

The temptation was to also clean up the socket file on exit, which sounds
obviously right and is where most of the exploration went. Go's `UnixListener`
already unlinks on `Close()` (verified empirically); the file survives only
because websocketd has *no* signal handling in `main()`, so SIGTERM kills it
before anything unwinds. (The SIGINT/SIGTERM logic in `process_endpoint.go` is
for child processes, unrelated.) So the fix would have been `signal.Notify` +
`listener.Close()`.

Deliberately not done. Installing a handler only on the `--unixsocket` path
made an identical SIGTERM exit 143 for a TCP-only server and 0 for a socket
one — an exit contract depending on an unrelated flag. Installing it globally
fixed that inconsistency but changed Ctrl+C and SIGTERM semantics for *every*
user (new exit code, new log lines) to serve a niche feature. Neither trade is
worth it: too much blast radius for the payoff. Socket files therefore outlive
their server, exactly as they did before, and startup recovery remains the
only cleanup mechanism.

That is also the more honest arrangement. Exit cleanup can never be complete —
SIGKILL, OOM, power loss all skip it — so startup recovery has to work
regardless, and adding a second partial mechanism would have implied a
guarantee that does not exist.

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

Unbounded inbound frame (#1): readFrames does io.ReadAll(rd) and
SetReadLimit was never called, so gorilla's default (unlimited) let one
client buffer an arbitrarily large message in memory. Added --maxframesize
(int64 bytes) plumbed to WebSocketEndpoint, which calls ws.SetReadLimit when
>0. Chose a finite 1 MiB *default* (0 disables) over preserving unlimited —
a user-visible behavior change, but the DoS was the default and the flag lets
large-frame users opt back out. Over-limit frames make NextReader return
ErrReadLimit; readFrames breaks and closes output, gorilla sends 1009.

Timeouts (#2): set ReadHeaderTimeout on all three servers. Deliberately NOT
ReadTimeout/WriteTimeout on the main server — those would kill long-lived
WebSocket and streaming-CGI responses. ReadHeaderTimeout only bounds the
header read before the handler runs and before the WS hijack, so it is safe
everywhere. The redirect server emits only tiny responses, so it gets the
full set.

TLS min version (#4) via a shared tlsConfig() helper — a one-liner.

#5/#6 revisited: initially documented-only, then changed after discussing what
sensible defaults should be. #5: flipped --maxforks default from 0 (unlimited)
to 1024. The default's job is a runaway backstop, not right-sizing — per-fork
cost is program-dependent (2 MB shell vs 40 MB Python), so no number is
"correct" as a capacity plan. 1024 protects the casual majority who never set
the flag (websocketd's whole ethos is one-liner deployments) while the
high-concurrency operators who'd notice are exactly the ones who set it
explicitly. 0 still means unlimited. Guarded by a test asserting the default
stays finite, since the mechanism itself is already tested.

#6: kept the scheme-agnostic match semantics (consistent with the existing
"unspecified port = any port" rule, and --origin's syntax is host-centric;
requiring a scheme would break every existing --origin=host config for a
narrow gain). Instead closed the sharp edge with a startup warning fired only
when --ssl is set AND a scheme-less --origin is given — the one configuration
where silently accepting http origins is surprising and the operator can act
on it.

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
