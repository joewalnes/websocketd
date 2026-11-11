// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cliflags defines every websocketd command line flag on a
// flag.FlagSet, without parsing argv, validating anything, or touching
// process state (no os.Exit).
//
// It exists so the exact same flag definitions that --help is generated
// from (name, default, usage string) can also be walked by tools/gendocs
// via FlagSet.VisitAll, to regenerate release/websocketd.man and
// docsite/content/reference/cli-flags.md. Go disallows importing a
// package named "main", so this registration logic could not stay inside
// config.go (package main) and still be reachable, in-process, from the
// tools/gendocs command — hence this package. config.go's
// parseCommandLine calls Register then does the actual Parse/validate/
// exit dance; tools/gendocs calls Register and only ever reads the
// result back with VisitAll.
package cliflags

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

// DefaultMaxForks is a finite runaway backstop, not a capacity plan. Each
// fork is a full subprocess, so unlimited (0) lets one client fork-bomb
// the host by opening connections. A casual deployment never legitimately
// needs this many concurrent long-lived connections; high-concurrency
// operators set --maxforks (or 0 for unlimited) explicitly. See DIARY
// 2026-08-17.
const DefaultMaxForks = 1024

// defaultPassEnv is borrowed from net/http/cgi.
var defaultPassEnv = map[string]string{
	"darwin":  "PATH,DYLD_LIBRARY_PATH",
	"freebsd": "PATH,LD_LIBRARY_PATH",
	"hpux":    "PATH,LD_LIBRARY_PATH,SHLIB_PATH",
	"irix":    "PATH,LD_LIBRARY_PATH,LD_LIBRARYN32_PATH,LD_LIBRARY64_PATH",
	"linux":   "PATH,LD_LIBRARY_PATH",
	"openbsd": "PATH,LD_LIBRARY_PATH",
	"solaris": "PATH,LD_LIBRARY_PATH,LD_LIBRARY_PATH_32,LD_LIBRARY_PATH_64",
	"windows": "PATH,SystemRoot,COMSPEC,PATHEXT,WINDIR",
}

// Arglist is a repeatable string flag value (flag.Value): each
// --flag=value occurrence appends to the slice, in the order given.
type Arglist []string

func (al *Arglist) String() string {
	return fmt.Sprintf("%v", []string(*al))
}

func (al *Arglist) Set(value string) error {
	*al = append(*al, value)
	return nil
}

// Set holds pointers to the value of every registered websocketd command
// line flag, plus the FlagSet they live on.
type Set struct {
	FS *flag.FlagSet

	Addrlist Arglist

	// server config options
	Port                   *int
	UnixSocket, SocketMode *string
	Version, License       *bool
	LogLevel               *string
	SSL                    *bool
	SSLCert, SSLKey, SSLCA *string
	MaxForks               *int
	CloseMs, PingMs        *uint
	MaxFrameSize           *int64
	RedirPort              *int

	// lib config options
	Binary, PassStderr    *bool
	Raw                   *bool
	ReverseLookup         *bool
	ScriptDir, StaticDir  *string
	CgiDir                *string
	DevConsole            *bool
	PassEnv               *string
	SameOrigin, AnyOrigin *bool
	AllowOrigins          *string

	Headers, HeadersWs, HeadersHttp Arglist
}

// Register defines every websocketd command line flag on a fresh FlagSet
// and returns pointers to their values. It does not call Parse and never
// exits, so it is safe to call from anything that only wants to
// introspect the flag definitions (name, default, usage) via
// FlagSet.VisitAll, without running any of websocketd's own startup,
// parsing, or validation logic.
//
// If adding a new command line option, also update the help text in
// help.go (the flag library's auto-generated help message isn't pretty
// enough to use directly) and, if it deserves prose beyond its bare
// --help usage string, the per-flag notes table in tools/gendocs.
func Register() *Set {
	var fv Set

	fv.FS = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fv.FS.Usage = func() {}

	fv.Addrlist = Arglist(make([]string, 0, 1)) // pre-reserve for 1 address
	fv.FS.Var(&fv.Addrlist, "address", "Interfaces to bind to (e.g. 127.0.0.1 or [::1]).")

	// server config options
	fv.Port = fv.FS.Int("port", 0, "HTTP port to listen on")
	fv.UnixSocket = fv.FS.String("unixsocket", "", "Path of a Unix domain socket to listen on, in addition to (or instead of) --address/--port")
	fv.SocketMode = fv.FS.String("socketmode", "", "Octal permission bits to force on the --unixsocket file (e.g. 0700); default follows umask")
	fv.Version = fv.FS.Bool("version", false, "Print version and exit")
	fv.License = fv.FS.Bool("license", false, "Print license and exit")
	fv.LogLevel = fv.FS.String("loglevel", "access", "Log level, one of: debug, trace, access, info, error, fatal, none")
	fv.SSL = fv.FS.Bool("ssl", false, "Use TLS on listening socket (see also --sslcert and --sslkey)")
	fv.SSLCert = fv.FS.String("sslcert", "", "Should point to certificate PEM file when --ssl is used")
	fv.SSLKey = fv.FS.String("sslkey", "", "Should point to certificate private key file when --ssl is used")
	fv.MaxForks = fv.FS.Int("maxforks", DefaultMaxForks, "Max forks, zero means unlimited")
	fv.CloseMs = fv.FS.Uint("closems", 0, "Extra time added to each of the first three waits before websocketd escalates to the next termination signal")
	fv.PingMs = fv.FS.Uint("pingms", 0, "WebSocket ping interval in milliseconds (0 disables)")
	fv.MaxFrameSize = fv.FS.Int64("maxframesize", 1<<20, "Max inbound WebSocket message size in bytes (0 = unlimited)")
	fv.RedirPort = fv.FS.Int("redirport", 0, "HTTP port to redirect to canonical --port address")
	fv.SSLCA = fv.FS.String("sslca", "", "CA certificate file for client certificate verification (mutual TLS)")

	// lib config options
	fv.Binary = fv.FS.Bool("binary", false, "Set websocketd to experimental binary mode (default is line by line)")
	fv.Raw = fv.FS.Bool("raw", false, "Set websocketd to experimental raw mode (like text mode but char by char instead of line by line)")
	fv.PassStderr = fv.FS.Bool("passstderr", false, "Forward STDERR to WebSocket clients as tagged JSON messages, alongside tagged STDOUT (mutually exclusive with --binary)")
	fv.ReverseLookup = fv.FS.Bool("reverselookup", false, "Perform reverse DNS lookups on remote clients")
	fv.ScriptDir = fv.FS.String("dir", "", "Base directory for WebSocket scripts")
	fv.StaticDir = fv.FS.String("staticdir", "", "Serve static content from this directory over HTTP")
	fv.CgiDir = fv.FS.String("cgidir", "", "Serve CGI scripts from this directory over HTTP")
	fv.DevConsole = fv.FS.Bool("devconsole", false, "Enable development console (cannot be used in conjunction with --staticdir or --cgidir)")
	fv.PassEnv = fv.FS.String("passenv", defaultPassEnv[runtime.GOOS], "List of envvars to pass to subprocesses (others will be cleaned out)")
	fv.SameOrigin = fv.FS.Bool("sameorigin", false, "Restrict upgrades if origin and host headers differ")
	fv.AnyOrigin = fv.FS.Bool("anyorigin", false, "Explicitly accept any origin (the current default) and silence the origin-policy startup warning")
	fv.AllowOrigins = fv.FS.String("origin", "", "Restrict upgrades if origin does not match the list")

	fv.Headers = Arglist(make([]string, 0))
	fv.HeadersWs = Arglist(make([]string, 0))
	fv.HeadersHttp = Arglist(make([]string, 0))
	fv.FS.Var(&fv.Headers, "header", "Custom headers for any response.")
	fv.FS.Var(&fv.HeadersWs, "header-ws", "Custom headers for successful WebSocket upgrade responses.")
	fv.FS.Var(&fv.HeadersHttp, "header-http", "Custom headers for all but WebSocket upgrade HTTP responses.")

	return &fv
}
