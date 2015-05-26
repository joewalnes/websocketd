// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	help = `
{{binary}} ({{version}})

{{binary}} is a command line tool that will allow any executable program
that accepts input on stdin and produces output on stdout to be turned into
a WebSocket server.

Usage:

  Export a single executable program a WebSocket server:
    {{binary}} [options] COMMAND [command args]

  Or, export an entire directory of executables as WebSocket endpoints:
    {{binary}} [options] --dir=SOMEDIR

Options:

  --port=PORT                    HTTP port to listen on.

  --address=ADDRESS              Address to bind to (multiple options allowed)
                                 Use square brackets to specify IPv6 address. 
                                 Default: "" (all)

  --sameorigin={true,false}      Restrict (HTTP 403) protocol upgrades if the
                                 Origin header does not match to requested HTTP 
                                 Host. Default: false.

  --origin=host[:port][,host[:port]...]
                                 Restrict (HTTP 403) protocol upgrades if the
                                 Origin header does not match to one of the host
                                 and port combinations listed. If the port is not
                                 specified, any port number will match. 
                                 Default: "" (allow any origin)

  --ssl                          Listen for HTTPS socket instead of HTTP.                     
  --sslcert=FILE                 All three options must be used or all of
  --sslkey=FILE                  them should be omitted. 

  --passenv VAR[,VAR...]         Lists environment variables allowed to be
                                 passed to executed scripts.

  --reverselookup={true,false}   Perform DNS reverse lookups on remote clients.
                                 Default: true

  --dir=DIR                      Allow all scripts in the local directory
                                 to be accessed as WebSockets. If using this,
                                 option, then the standard program and args
                                 options should not be specified.

  --staticdir=DIR                Serve static files in this directory over HTTP.

  --cgidir=DIR                   Serve CGI scripts in this directory over HTTP.

  --header="..."                 Set custom HTTP header to each answer. For
                                 example: --header="Server: someserver/0.0.1"

  --header-ws="...."             Same as --header, just applies to only those
                                 responses that indicate upgrade of TCP connection
                                 to a WebSockets protocol.

  --header-http="...."           Same as --header, just applies to only to plain
                                 HTTP responses that do not indicate WebSockets
                                 upgrade


  --help                         Print help and exit.

  --version                      Print version and exit.

  --license                      Print license and exit.

  --devconsole                   Enable interactive development console.
                                 This enables you to access the websocketd
                                 server with a web-browser and use a
                                 user interface to quickly test WebSocket
                                 endpoints. For example, to test an
                                 endpoint at ws://[host]/foo, you can
                                 visit http://[host]/foo in your browser.
                                 This flag cannot be used in conjunction
                                 with --staticdir or --cgidir.

  --loglevel=LEVEL               Log level to use (default access).
                                 From most to least verbose:
                                 debug, trace, access, info, error, fatal

Full documentation at http://websocketd.com/

Copyright 2013 Joe Walnes and the websocketd team. All rights reserved.
BSD license: Run '{{binary}} --license' for details.
`
	short = `
Usage:

  Export a single executable program a WebSocket server:
    {{binary}} [options] COMMAND [command args]

  Or, export an entire directory of executables as WebSocket endpoints:
    {{binary}} [options] --dir=SOMEDIR

  Or, show extended help message using:
    {{binary}} --help
`
)

func get_help_message(content string) string {
	msg := strings.Trim(content, " \n")
	msg = strings.Replace(msg, "{{binary}}", HelpProcessName(), -1)
	return strings.Replace(msg, "{{version}}", Version(), -1)
}

func HelpProcessName() string {
	binary := os.Args[0]
	if strings.Contains(binary, "/go-build") { // this was run using "go run", let's use something appropriate
		binary = "websocketd"
	} else {
		binary = filepath.Base(binary)
	}
	return binary
}

func PrintHelp() {
	fmt.Fprintf(os.Stderr, "%s\n", get_help_message(help))
}

func ShortHelp() {
	// Shown after some error
	fmt.Fprintf(os.Stderr, "\n%s\n", get_help_message(short))
}
