// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"time"
)

type Config struct {
	// base initiaization fields
	StartupTime    time.Time // Server startup time (used for dev console caching).
	CommandName    string    // Command to execute.
	CommandArgs    []string  // Additional args to pass to command.
	ServerSoftware string    // Value to pass to SERVER_SOFTWARE environment variable (e.g. websocketd/1.2.3).

	// settings
	ReverseLookup  bool     // Perform reverse DNS lookups on hostnames (useful, but slower).
	Ssl            bool     // websocketd works with --ssl which means TLS is in use
	ScriptDir      string   // Base directory for websocket scripts.
	UsingScriptDir bool     // Are we running with a script dir.
	StaticDir      string   // If set, static files will be served from this dir over HTTP.
	CgiDir         string   // If set, CGI scripts will be served from this dir over HTTP.
	DevConsole     bool     // Enable dev console. This disables StaticDir and CgiDir.
	AllowOrigins   []string // List of allowed origin addresses for websocket upgrade.
	SameOrigin     bool     // If set, requires websocket upgrades to be performed from same origin only.
	Headers        []string
	HeadersWs      []string
	HeadersHTTP    []string

	// created environment
	Env       []string // Additional environment variables to pass to process ("key=value").
	ParentEnv []string // Variables kept from os.Environ() before sanitizing it for subprocess.
}
