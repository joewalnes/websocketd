// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"time"
)

type Config struct {
	CommandName    string    // Command to execute.
	CommandArgs    []string  // Additional args to pass to command.
	ReverseLookup  bool      // Perform reverse DNS lookups on hostnames (useful, but slower).
	ScriptDir      string    // Base directory for websocket scripts.
	UsingScriptDir bool      // Are we running with a script dir.
	StartupTime    time.Time // Server startup time (used for dev console caching).
	DevConsole     bool      // Enable dev console.
	ServerSoftware string    // Value to pass to SERVER_SOFTWARE environment variable (e.g. websocketd/1.2.3).
	Env            []string  // Additional environment variables to pass to process ("key=value").
}
