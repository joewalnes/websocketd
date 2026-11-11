// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command gendocs regenerates the flag-derived parts of
// release/websocketd.man and docsite/content/reference/cli-flags.md from
// the same live flag definitions --help itself is rendered from (see
// internal/cliflags). Run it after adding, removing, or changing any
// command line flag:
//
//	go run ./tools/gendocs
//
// It writes both files in place; review and commit the diff like any
// other generated artifact. CI re-runs this exact command and fails the
// build if the working tree then differs from what's committed, so the
// flag table in either file can never silently drift from the flags
// websocketd actually accepts (see .github/workflows/docsdrift.yml).
//
// --help is not itself registered on the FlagSet (Go's flag package
// intercepts -h/--help automatically), so it is added to the generated
// output as one manually-declared entry below, alongside the real,
// VisitAll-derived ones.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/joewalnes/websocketd/internal/cliflags"
)

const manPath = "release/websocketd.man"
const cliFlagsDir = "docsite/content/reference"
const cliFlagsMdPath = cliFlagsDir + "/cli-flags.md"

// flagDoc is a generation-friendly description of one flag, built either
// from a real *flag.Flag (via VisitAll) or, for --help, by hand.
type flagDoc struct {
	Name     string
	Usage    string
	Default  string // display form, already computed
	Bool     bool   // rendered bare, e.g. --devconsole, not --devconsole=VALUE
	Repeated bool   // backed by cliflags.Arglist: may be given multiple times
}

// boolFlag mirrors the unexported interface the flag package itself uses
// internally (flag.Value plus IsBoolFlag() bool) to tell a boolean flag
// from a string/int/etc one, so bool flags render bare (--devconsole)
// rather than with a placeholder value (--devconsole=VALUE).
type boolFlag interface {
	IsBoolFlag() bool
}

// flagNotes carries hand-written prose that goes beyond a flag's bare
// --help usage string: the "why", the gotcha, the issue number. It is
// intentionally a small hand-maintained table, keyed by flag name — but
// the existence, name, and default of every flag still comes from
// cliflags.Register()/VisitAll, so a flag can never be silently dropped
// from the reference, or documented with a stale default, just because
// this table wasn't updated. A flag absent from this table still gets a
// full entry, just without the extra paragraph.
//
// See docs-archive/MANUAL_INVENTORY.md §1 for the sourcing behind each
// note (issue numbers included so a reader can find the original report).
var flagNotes = map[string]string{
	"address": "May be given more than once to bind several interfaces. Every " +
		"address listens on the same --port.",

	"binary": "Switches the WebSocket message type to binary frames instead of text, " +
		"and removes the newline framing rule: output is forwarded in raw chunks as " +
		"read, with no newline required, and no newline is appended to input. It does " +
		"not allocate a pseudo-terminal, so a program that only behaves interactively " +
		"under a terminal still does not do so here (issue #443). See [message " +
		"framing](/understanding/message-framing/).",

	"cgidir": "Scripts here run as CGI programs answering ordinary HTTP requests, not " +
		"WebSocket upgrades. The request path must name the script file exactly; there " +
		"is no extra path information after it. When the CGI directory sits inside the " +
		"--staticdir tree, its scripts also answer at their path there: with " +
		"--staticdir=/PAGE --cgidir=/PAGE/cgi-bin, /cgi-bin/hello.sh runs the same " +
		"script as /hello.sh. That position is decided by which directory each flag " +
		"names rather than by how it is spelled, so a symlinked or differently-cased " +
		"pair of paths routes the same way, and a --cgidir genuinely outside the static " +
		"tree is not brought inside it by a symlink there. It is worked out once and " +
		"reused, so a deployment symlink flipped under a running server does not " +
		"re-route requests; restart to pick up a new layout. Files in the CGI directory " +
		"are never served as static content. The variables a CGI script receives " +
		"differ from the WebSocket ones, because Go's net/http/cgi builds them: see " +
		"[environment variables](/reference/environment-variables/). May be combined " +
		"with --dir, which serves a separate directory over WebSocket.",

	"closems": "The value is added to each of the first three steps of the teardown " +
		"ladder: closing stdin (100 ms), SIGINT (250 ms), and SIGTERM (500 ms). The " +
		"final SIGKILL step waits a fixed 1000 ms and is not affected. Signals are sent " +
		"whatever the value; 0 means no extra delay, not no signals. See [process " +
		"lifecycle](/understanding/process-lifecycle/).",

	"devconsole": "All three of --devconsole, --staticdir and --cgidir claim the same " +
		"non-WebSocket HTTP surface, so websocketd exits with code 4 if the console is " +
		"combined with either of the others. See [dev " +
		"console](/reference/dev-console/).",

	"dir": "Any file under this directory is reachable as its own WebSocket endpoint, " +
		"named after its path within the directory. The matched path becomes " +
		"SCRIPT_NAME and anything left over becomes PATH_INFO. The executable bit is " +
		"not checked when the path is resolved; whether the file runs is decided when " +
		"websocketd launches it. A file reached through a symlink that leaves the " +
		"directory is answered with 404. Files here are never served as static " +
		"content, so a plain GET of a script that also sits inside a --staticdir tree " +
		"returns 404 rather than the script's source. A script keeps the URL --dir " +
		"gives it and gains no second one from where the directory sits: unlike " +
		"--cgidir, no URL prefix is derived for it inside --staticdir. Giving both " +
		"--dir and a COMMAND is rejected at startup. May be combined with --cgidir, " +
		"which serves a separate directory as CGI over HTTP.",

	"header": "May be given more than once. Added to successful WebSocket upgrade " +
		"responses and to every response the WebSocket handler did not produce: the dev " +
		"console, CGI, static files, and 404s. Error responses from the WebSocket " +
		"handler itself, such as a rejected upgrade or a 429, carry no configured " +
		"headers.",

	"header-http": "May be given more than once. Added to every response the WebSocket " +
		"handler did not produce: the dev console, CGI, static files, and 404s.",

	"header-ws": "May be given more than once. Added to successful WebSocket upgrade " +
		"responses only.",

	"maxforks": "Each WebSocket connection and each CGI request is a full subprocess, " +
		"and this caps how many may be live at once. Beyond the cap, an upgrade or CGI " +
		"request is answered with 429 Too Many Requests rather than queued. Zero means " +
		"unlimited. Operators running many concurrent long-lived connections have hit " +
		"the default ceiling (issues #226, #228, #356). See [the process " +
		"model](/understanding/process-model/).",

	"maxframesize": "An inbound WebSocket message larger than this limit is rejected " +
		"and the connection is closed with status 1009 (message too big), which bounds " +
		"how much one client can make websocketd buffer. Zero disables the limit. A " +
		"negative value is rejected at startup with exit code 1, because it would " +
		"otherwise read as unlimited and silently remove the limit (issue #472). See " +
		"[security model](/understanding/security-model/).",

	"origin": "Entries are comma-separated. An entry with no scheme (just host[:port]) " +
		"matches both http and https origins; an entry prefixed \"https://\" matches " +
		"https only. An entry with an explicit port matches that port only. An entry " +
		"with no port matches only the default port of a scheme it accepts, 80 for http " +
		"and 443 for https. Appending \":*\" to an entry matches any port on that host " +
		"(issue #473). Combining this flag with --anyorigin is rejected at startup. See " +
		"[security model](/understanding/security-model/).",

	"anyorigin": "States the current permissive-origin default explicitly and silences " +
		"the origin-policy warning websocketd prints to stderr at startup. Combining it " +
		"with --sameorigin or --origin is rejected at startup, since those restrict what " +
		"this flag accepts. A future websocketd release defaults to --sameorigin " +
		"instead. See [security model](/understanding/security-model/).",

	"sameorigin": "Accepts an upgrade only when the host and port in the request's " +
		"Origin header match the host and port in its Host header. A request with no " +
		"Origin header is treated as origin \"file:\" and does not match. Combining it " +
		"with --anyorigin is rejected at startup. See [security " +
		"model](/understanding/security-model/).",

	"passenv": "The default is PATH plus the platform's shared library search path: " +
		"PATH,LD_LIBRARY_PATH on Linux, PATH,DYLD_LIBRARY_PATH on macOS, and " +
		"PATH,SystemRoot,COMSPEC,PATHEXT,WINDIR on Windows. Passing --passenv REPLACES " +
		"that default rather than adding to it, so --passenv=API_KEY alone leaves the " +
		"child with no PATH. A named variable that is empty or unset in websocketd's own " +
		"environment is dropped, not forwarded as an empty string. HTTPS is always " +
		"skipped, because that variable is websocketd's own --ssl signal. This flag " +
		"controls only which of websocketd's OWN environment variables reach the child; " +
		"per-request CGI variables such as QUERY_STRING are built separately and are " +
		"unaffected by it, a recurring point of confusion (issues #202, #223, #312, " +
		"#391). See [passing data into your " +
		"script](/how-to/patterns/pass-arguments/).",

	"passstderr": "The child's stdout and stderr both reach the client as JSON objects " +
		"of the form {\"stream\":\"stdout\",\"data\":\"...\"}, one per line of output, " +
		"so the two streams can be told apart. Plain (untagged) output is no longer " +
		"sent. stderr is still written to websocketd's own log as well. Combining it " +
		"with --binary is rejected at startup with exit code 1.",

	"pingms": "Zero disables pings entirely, and an idle connection is then never " +
		"timed out; this is the default, and it accounts for a long run of " +
		"\"mystery disconnect\" reports where the connection was in fact dropped by " +
		"something in between (issues #37, #209, #260, #275, #439). A non-zero value " +
		"sends a WebSocket ping every interval and sets a read deadline of twice the " +
		"interval. Only an incoming pong resets that deadline, so a client that keeps " +
		"sending data but never answers a ping is disconnected within 2x --pingms just " +
		"the same. See [process " +
		"lifecycle](/understanding/process-lifecycle/).",

	"port": "The default of 0 is not a real port: it means 80, or 443 when --ssl is " +
		"given. Every --address binds this same port.",

	"redirport": "Runs a second, plain HTTP listener on this port whose only response " +
		"is a 301 redirect to the main listener. The redirect target keeps the host the " +
		"client itself sent, along with the path and query it asked for, and rewrites " +
		"only the scheme and the port, so a link into the site arrives at that link and " +
		"it is not an open redirect.",

	"reverselookup": "Sets REMOTE_HOST to the result of a reverse DNS lookup of the " +
		"client's address instead of the address itself. The lookup happens on every " +
		"connection. It has no effect on --cgidir requests, where net/http/cgi sets " +
		"REMOTE_HOST to the client IP regardless. See [environment " +
		"variables](/reference/environment-variables/).",

	"socketmode": "Applied with chmod to the socket file immediately after bind, so " +
		"the window in which the umask-derived mode applies is as short as it can be. " +
		"An empty value leaves the mode to the process umask; websocketd does not " +
		"change the default socket permissions (issue #474). A value of 0, a value " +
		"above 0777, and a value that is not octal are each rejected at startup with " +
		"exit code 1. It applies only to --unixsocket.",

	"ssl": "Requires both --sslcert and --sslkey; giving --ssl without them, or " +
		"either of them without --ssl, is rejected at startup with exit code 1. The " +
		"listener negotiates TLS 1.2 or higher. The spawned process additionally " +
		"receives HTTPS=on. See [serve over " +
		"wss://](/how-to/deploy/tls/).",

	"sslca": "Turns on mutual TLS: client certificates are required and verified " +
		"against this CA file. It requires --ssl; giving --sslca without --ssl is " +
		"rejected at startup with exit code 1, because mutual TLS has no handshake " +
		"to verify a client certificate in without a TLS listener (issue #477). See " +
		"[serve over wss://](/how-to/deploy/tls/).",

	"sslcert": "Read only when --ssl is given. Giving it without --ssl is rejected at " +
		"startup with exit code 1.",

	"sslkey": "Read only when --ssl is given. Giving it without --ssl is rejected at " +
		"startup with exit code 1.",

	"staticdir": "Four kinds of request are answered with 404 rather than served: a " +
		"path with a segment beginning with \".\" (.git, .env, .ssh, ...); a directory " +
		"with no index.html file, since directories are never listed; a file reached " +
		"through a symlink that leaves the directory; and any file inside a --dir or " +
		"--cgidir tree, whichever URL reaches it. That last refusal is dropped when " +
		"--staticdir names one of those directories itself, or a directory inside one. " +
		"A first path segment of exactly .well-known is exempt from the dotfile rule, " +
		"so ACME challenges and security.txt are still served; a dotfile nested deeper " +
		"inside it is not. None of that vets the directory's contents: a secret under " +
		"an ordinary name is still served, so point --staticdir at a tree holding only " +
		"what is meant to be public. Rejected together with --devconsole. Why each of " +
		"these is refused is in the [security " +
		"model](/understanding/security-model/).",

	"unixsocket": "Served in addition to the TCP listener. When --unixsocket is the " +
		"only listening flag given, with no --port, --address, or --redirport, no TCP " +
		"listener is started at all. At startup a socket file already at the path is " +
		"probed: if nothing is listening on it, it is removed as stale and rebound; if " +
		"something is, websocketd exits with code 3 rather than making the running " +
		"server unreachable. The file is not removed on exit.",
}

// defaultOverrides replaces a flag's computed default display where the real
// one is not stable across platforms. runtime.GOOS decides --passenv's default,
// so rendering flag.DefValue verbatim bakes the generating machine's OS into
// both generated files: regenerating on Linux and on macOS produce different
// bytes, and CI (which runs the generator on Linux and diffs) fails against a
// tree generated anywhere else. The per-OS values live in the flagNotes prose
// instead, where they can all be stated at once.
var defaultOverrides = map[string]string{
	"passenv": "platform-dependent",
}

// valueHint returns the placeholder shown after "=" for a non-boolean
// flag, e.g. "--port=PORT". It is derived mechanically from the flag's
// own name (upper-cased) rather than a hand-maintained table, so it can
// never fall out of sync with the flags that actually exist.
func valueHint(name string) string {
	return strings.ToUpper(name)
}

func main() {
	docs := collectFlagDocs()

	if err := os.WriteFile(manPath, []byte(renderMan(docs)), 0644); err != nil { // #nosec G306
		fmt.Fprintf(os.Stderr, "gendocs: writing %s: %s\n", manPath, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cliFlagsDir, 0755); err != nil { // #nosec G301
		fmt.Fprintf(os.Stderr, "gendocs: creating %s: %s\n", cliFlagsDir, err)
		os.Exit(1)
	}
	if err := os.WriteFile(cliFlagsMdPath, []byte(renderCliFlagsMd(docs)), 0644); err != nil { // #nosec G306
		fmt.Fprintf(os.Stderr, "gendocs: writing %s: %s\n", cliFlagsMdPath, err)
		os.Exit(1)
	}

	fmt.Printf("gendocs: wrote %s and %s (%d flags)\n", manPath, cliFlagsMdPath, len(docs))
}

// collectFlagDocs walks the real, live flag definitions via
// cliflags.Register()+VisitAll — never parsing argv, never exiting — and
// adds the one synthetic entry for --help, which the flag package
// handles specially and never registers as a real *flag.Flag.
func collectFlagDocs() []flagDoc {
	fv := cliflags.Register()

	var docs []flagDoc
	fv.FS.VisitAll(func(f *flag.Flag) {
		docs = append(docs, flagDocFrom(f))
	})

	docs = append(docs, flagDoc{
		Name:    "help",
		Usage:   "Print help and exit.",
		Default: "false",
		Bool:    true,
	})

	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	return docs
}

// flagDocFrom converts one real *flag.Flag (from VisitAll) into a
// flagDoc, computing a human-friendly default display and detecting
// whether the flag is boolean or repeatable (cliflags.Arglist-backed).
func flagDocFrom(f *flag.Flag) flagDoc {
	d := flagDoc{Name: f.Name, Usage: f.Usage}

	if _, ok := f.Value.(*cliflags.Arglist); ok {
		d.Repeated = true
		d.Default = "none (may be given multiple times)"
		return d
	}

	if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
		d.Bool = true
	}

	if override, ok := defaultOverrides[f.Name]; ok {
		d.Default = override
		return d
	}

	if f.DefValue == "" {
		d.Default = `"" (empty)`
	} else {
		d.Default = f.DefValue
	}
	return d
}

// usageSentence returns f.Usage with a trailing period, so entries read
// as complete sentences regardless of how the underlying usage string in
// internal/cliflags happened to be punctuated.
func usageSentence(usage string) string {
	usage = strings.TrimSpace(usage)
	if usage == "" {
		return usage
	}
	if !strings.HasSuffix(usage, ".") && !strings.HasSuffix(usage, "!") && !strings.HasSuffix(usage, "?") {
		usage += "."
	}
	return usage
}

// docsBaseURL is where the rendered docsite lives. The per-flag notes carry
// root-relative markdown links, which mean nothing in a man page, so the man
// renderer expands them to absolute URLs against this base.
const docsBaseURL = "https://websocketd.com/docs"

// mdLink matches one inline markdown link, [text](/path/).
var mdLink = regexp.MustCompile(`\[([^\]]+)\]\((/[^)]*)\)`)

// expandMarkdownLinks rewrites [text](/path/) as "text (https://.../path/)",
// so a note written for the markdown page still reads as a sentence, with a
// usable address, in the man page.
func expandMarkdownLinks(s string) string {
	return mdLink.ReplaceAllString(s, "$1 ("+docsBaseURL+"$2)")
}

// manEscape escapes the one troff metacharacter sequence that shows up in
// our flag names and usage text: a literal "--", which needs to be
// "\-\-" so it renders as two hyphens instead of an en dash. Single
// hyphens (e.g. inside "HTTPS-only") are left alone, matching this man
// page's existing convention.
func manEscape(s string) string {
	return strings.ReplaceAll(s, "--", `\-\-`)
}

const manHeader = `.\" Manpage for websocketd.
.\" Contact abc@alexsergeyev.com to correct errors or typos.
.TH websocketd 1 "9 Jul 2026" "0.5.0" "websocketd man page"
.SH NAME
websocketd \- turns any program that uses STDIN/STDOUT into a WebSocket server.
.SH SYNOPSIS
websocketd [options] COMMAND [command args]

or

websocketd [options] --dir=SOMEDIR
.SH DESCRIPTION
\fBwebsocketd\fR is a command line tool that will allow any executable program
that accepts input on stdin and produces output on stdout to be turned into
a WebSocket server.

To learn more about websocketd visit \fIhttps://websocketd.com\fR and project WIKI
on GitHub!
.SH OPTIONS
A summary of the options supported by websocketd is included below.
.PP
`

const manFooter = `.SH SEE ALSO
.RS 2
* full documentation at \fIhttps://websocketd.com\fR
.RE
.RS 2
* project source at \fIhttps://github.com/joewalnes/websocketd\fR
.RE
.SH BUGS
The only known condition so far is that certain applications in programming languages that enforce implicit STDOUT buffering (Perl, Python, etc.) would be producing unexpected data passing
delays when run under \fBwebsocketd\fR. Such issues could be solved by editing the source code of those applications (prohibiting buffering) or modifying their environment to trick them
into autoflush mode (e.g. pseudo-terminal wrapper "unbuffer").

Active issues in development are discussed on GitHub: \fIhttps://github.com/joewalnes/websocketd/issues\fR.

Please use that page to share your concerns and ideas about \fBwebsocketd\fR, authors would greatly appreciate your help!
.SH AUTHOR
Copyright 2013-2014 Joe Walnes and the websocketd team. All rights reserved.

BSD license: Run 'websocketd \-\-license' for details.
`

// renderMan rebuilds release/websocketd.man in full: the NAME, SYNOPSIS,
// DESCRIPTION, OPTIONS intro, SEE ALSO, BUGS, and AUTHOR sections are
// static (manHeader/manFooter, copied verbatim from the file as it stood
// before this generator existed) — only the flag table body in between
// is derived from the live flag definitions.
func renderMan(docs []flagDoc) string {
	var b strings.Builder
	b.WriteString(manHeader)

	for _, d := range docs {
		name := manEscape("--" + d.Name)
		if !d.Bool && !d.Repeated {
			name += "=" + valueHint(d.Name)
		}

		b.WriteString(name)
		b.WriteString("\n.RS 4\n")
		b.WriteString(manEscape(usageSentence(d.Usage)))
		if d.Default != "" {
			b.WriteString(" Default: ")
			b.WriteString(manEscape(d.Default))
			b.WriteString(".")
		}
		if note, ok := flagNotes[d.Name]; ok {
			b.WriteString(" ")
			b.WriteString(manEscape(expandMarkdownLinks(note)))
		}
		b.WriteString("\n.RE\n.PP\n")
	}

	// The flag loop always leaves a trailing ".PP\n" that belongs to the
	// *next* entry; the footer starts its own section header instead, so
	// drop that dangling ".PP\n" before appending it.
	out := b.String()
	out = strings.TrimSuffix(out, ".PP\n")
	out += manFooter
	return out
}

const cliFlagsFrontMatter = `---
title: "CLI flags"
weight: 10
description: "Every websocketd command line flag, its default, and exactly what it does."
---

`

const cliFlagsIntro = `Every flag websocketd accepts, in alphabetical order, with its default and its
exact effect. Order on the command line matters: see [flag placement](/reference/).

For a bare list with no notes, run ` + "`websocketd --help`" + `.

## Flags

`

// cliFlagsProvenance is the one provenance line the page carries, at the
// foot rather than the top: a reader who has scrolled the whole flag table
// is the one who might edit the file, and the reader who arrived for a
// default is not. STYLE.md section 3 forbids opening a page by describing
// the documentation.
const cliFlagsProvenance = "Generated from websocketd's flag definitions by `tools/gendocs`. " +
	"Edit the generator, not this page.\n"

// renderCliFlagsMd builds docsite/content/reference/cli-flags.md: Hugo
// front matter, a short intro (including the one prominent flag-ordering
// callout that issues #151/#155 asked for, given once here rather than
// repeated on every flag), then one section per flag in alphabetical
// order.
func renderCliFlagsMd(docs []flagDoc) string {
	var b strings.Builder
	b.WriteString(cliFlagsFrontMatter)
	b.WriteString(cliFlagsIntro)

	for _, d := range docs {
		flag := "--" + d.Name
		if !d.Bool && !d.Repeated {
			flag += "=" + valueHint(d.Name)
		}

		fmt.Fprintf(&b, "### `%s`\n\n", flag)
		fmt.Fprintf(&b, "**Default:** `%s`\n\n", d.Default)
		b.WriteString(usageSentence(d.Usage))
		b.WriteString("\n\n")
		if note, ok := flagNotes[d.Name]; ok {
			b.WriteString(note)
			b.WriteString("\n\n")
		}
	}

	out := strings.TrimRight(b.String(), "\n") + "\n\n"
	return out + cliFlagsProvenance
}
