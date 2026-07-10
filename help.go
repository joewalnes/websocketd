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

  --unixsocket=PATH              Path of a Unix domain socket to listen on,
                                 in addition to (or instead of) --address/
                                 --port. If it's the only listen option given
                                 (no --port, --address, or --redirport), no
                                 TCP listener is started at all. A leftover
                                 socket file from an unclean shutdown at the
                                 same path is removed automatically.
                                 Default: "" (do not listen on a Unix socket)

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

  --sslca=FILE                   Require clients to present a certificate
                                 signed by this CA (mutual TLS). Only takes
                                 effect together with --ssl.

  --redirport=PORT               Open alternative port and redirect HTTP traffic
                                 from it to canonical address (mostly useful
                                 for HTTPS-only configurations to redirect HTTP
                                 traffic)

  --passenv VAR[,VAR...]         Lists environment variables allowed to be
                                 passed to executed scripts. Does not work for
                                 Windows since all the variables are kept there.

  --binary={true,false}          Switches communication to binary, process reads
                                 send to browser as blobs and all reads from the
                                 browser are immediately flushed to the process.
                                 Default: false

  --passstderr                   Forward the process's STDERR to WebSocket
                                 clients, tagged (alongside STDOUT) as JSON:
                                 {"stream":"stdout","data":"..."} or
                                 {"stream":"stderr","data":"..."}. STDERR is
                                 still logged server-side either way. Cannot
                                 be combined with --binary. Default: false

  --reverselookup={true,false}   Perform DNS reverse lookups on remote clients.
                                 Default: false

  --dir=DIR                      Allow all scripts in the local directory
                                 to be accessed as WebSockets. If using this,
                                 option, then the standard program and args
                                 options should not be specified.

  --staticdir=DIR                Serve static files in this directory over HTTP.

  --cgidir=DIR                   Serve CGI scripts in this directory over HTTP.

  --maxforks=N                   Limit number of processes that websocketd is
                                 able to execute with WS and CGI handlers.
                                 When maxforks reached the server will be
                                 rejecting requests that require executing
                                 another process (unlimited when 0 or negative).
                                 Default: 0

  --closems=milliseconds         Specifies additional time process needs to gracefully
                                 finish before websocketd will send termination signals
                                 to it. Default: 0 (signals sent after 100ms, 250ms,
                                 and 500ms of waiting)

  --pingms=milliseconds          Send WebSocket pings at this interval and drop
                                 connections that miss pongs for twice that long,
                                 detecting dead clients. Default: 0 (disabled)

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

func helpMessage(content string) string {
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
	fmt.Fprintf(os.Stderr, "%s\n", helpMessage(help))
}

func ShortHelp() {
	// Shown after some error
	fmt.Fprintf(os.Stderr, "\n%s\n", helpMessage(short))
}
