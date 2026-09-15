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

  Export a single executable program as a WebSocket server:
    {{binary}} [options] COMMAND [command args]

  Or, export an entire directory of executables as WebSocket endpoints:
    {{binary}} [options] --dir=SOMEDIR

Options:

  --port=PORT                    HTTP port to listen on.
                                 Default: 80 (443 with --ssl)

  --address=ADDRESS              Address to bind to (multiple options allowed)
                                 Use square brackets to specify IPv6 address.
                                 Default: "" (all)

  --unixsocket=PATH              Path of a Unix domain socket to listen on,
                                 in addition to (or instead of) --address/
                                 --port. If it's the only listen option given
                                 (no --port, --address, or --redirport), no
                                 TCP listener is started at all. A leftover
                                 socket file from an unclean shutdown at the
                                 same path is removed automatically, but a
                                 path another server is still listening on is
                                 left alone and websocketd will not start.
                                 Default: "" (do not listen on a Unix socket)

  --socketmode=mode              Octal permission bits to force on the Unix
                                 socket file (e.g. 0700), applied immediately
                                 after binding. Without it the file follows the
                                 process umask, which can leave the socket
                                 connectable by other local users.
                                 Default: umask.

  --sameorigin={true,false}      Restrict (HTTP 403) protocol upgrades unless
                                 the Origin header matches the requested HTTP
                                 Host. A request that sends no Origin header
                                 does not match. Default: false.

  --anyorigin={true,false}       Explicitly accept upgrades from any origin,
                                 which is the current default behavior, and
                                 silence the origin-policy startup warning.
                                 Cannot be combined with --sameorigin or
                                 --origin. Note: a future version of websocketd
                                 will default to --sameorigin; pass --anyorigin
                                 to keep this behavior.
                                 Default: false.

  --origin=[scheme://]host[:port][,...]
                                 Restrict (HTTP 403) protocol upgrades if the
                                 Origin header does not match one of the listed
                                 hosts. A port, when given, must match exactly;
                                 an entry without a port matches only that
                                 scheme's default port (80 for http, 443 for
                                 https). Append ":*" to an entry to match any
                                 port. An entry without a scheme matches BOTH
                                 http and https origins; prefix "https://" to
                                 require TLS. Default: "" (allow any origin)

  --ssl                          Listen for HTTPS socket instead of HTTP.
  --sslcert=FILE                 All three options must be used or all of
  --sslkey=FILE                  them should be omitted.

  --sslca=FILE                   Require clients to present a certificate
                                 signed by this CA (mutual TLS). Requires
                                 --ssl; on its own it is rejected at startup.

  --redirport=PORT               Open alternative port and 301-redirect HTTP
                                 traffic from it to the canonical address
                                 (mostly useful for HTTPS-only configurations).
                                 Keeps the client's own host, path and query;
                                 only the scheme and port change.

  --passenv VAR[,VAR...]         Environment variables to pass to executed
                                 scripts. Replaces the default list rather than
                                 adding to it, so name every variable you want,
                                 including PATH. A variable that is unset or
                                 empty in websocketd's own environment is not
                                 passed on. Default: PATH plus the platform's
                                 library search path.

  --binary={true,false}          Use binary WebSocket frames and drop the
                                 newline framing: process output is forwarded
                                 in chunks as it is read, and no newline is
                                 appended to what the browser sends.
                                 Default: false

  --raw={true,false}             Switches text communication to raw mode, process
                                 read byte by byte and passes it to the counterpart
                                 without adding or removing \n to the messages.
                                 Please mote that, when using --passstderr, stderr
                                 is still handled on a line by line basis.
                                 Default: false

  --passstderr                   Forward the process's STDERR to WebSocket
                                 clients, tagged (alongside STDOUT) as JSON:
                                 {"stream":"stdout","data":"..."} or
                                 {"stream":"stderr","data":"..."}. STDERR is
                                 still logged server-side either way. Cannot
                                 be combined with --binary. Default: false

  --reverselookup={true,false}   Set REMOTE_HOST from a reverse DNS lookup of
                                 the client address instead of the address
                                 itself. Default: false

  --dir=DIR                      Serve every file under this directory as its
                                 own WebSocket endpoint, named by its path
                                 within the directory. Files here are never
                                 served as static content, unless --staticdir
                                 names this directory or one inside it. Cannot
                                 be combined with COMMAND.

  --staticdir=DIR                Serve static files in this directory over HTTP.
                                 Directories are never listed. Dotfiles and
                                 symlinks pointing out of the directory are
                                 refused, as are files inside a --dir or
                                 --cgidir tree.

  --cgidir=DIR                   Serve CGI scripts in this directory over HTTP.
                                 The request path names the script within the
                                 directory, so /hello.sh runs DIR/hello.sh.
                                 If DIR is inside --staticdir, its path there
                                 also works: with --staticdir=/PAGE
                                 --cgidir=/PAGE/cgi-bin, /cgi-bin/hello.sh
                                 runs the same script. Files in DIR are never
                                 served as static content.

  --maxforks=N                   Limit number of processes that websocketd is
                                 able to execute with WS and CGI handlers.
                                 When maxforks reached the server will be
                                 rejecting requests that require executing
                                 another process (unlimited when 0 or negative).
                                 Default: 1024, a runaway backstop rather than a
                                 capacity plan. Raise it for high-concurrency
                                 deployments, or set 0 for unlimited. Note it
                                 gates only WS upgrades and CGI execs, not
                                 static/redirect requests.

  --maxframesize=bytes           Reject inbound WebSocket messages larger than
                                 this, closing the connection with status 1009
                                 (bounds per-client memory use). Set 0 to
                                 disable the limit; a negative value is
                                 rejected at startup.
                                 Default: 1048576 (1 MiB)

  --closems=milliseconds         Extra time added to each of the first three
                                 waits when shutting a process down: stdin
                                 close (100ms), SIGINT (250ms), SIGTERM
                                 (500ms). The final SIGKILL wait of 1000ms is
                                 unaffected. Signals go to the process's whole
                                 process group; children that must outlive the
                                 session should start their own session
                                 (setsid). Default: 0 (no extra delay)

  --pingms=milliseconds          Send WebSocket pings at this interval and drop
                                 a connection that has not answered with a pong
                                 for twice that long. Only a pong resets that
                                 deadline; other traffic from the client does
                                 not. Default: 0 (no pings, no idle timeout)

  --header="..."                 Set custom HTTP header on each response. May
                                 be repeated. Error responses from the
                                 WebSocket handler (403, 404, 429) carry no
                                 configured headers. For example:
                                 --header="Server: someserver/0.0.1"

  --header-ws="...."             Same as --header, but only on responses that
                                 upgrade the connection to WebSocket.

  --header-http="...."           Same as --header, but only on plain HTTP
                                 responses that do not upgrade to WebSocket.

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
                                 debug, trace, access, info, error, fatal.
                                 Also accepts "none", which logs nothing.

Full documentation at https://websocketd.com/

Copyright 2013 Joe Walnes and the websocketd team. All rights reserved.
BSD license: Run '{{binary}} --license' for details.
`
	short = `
Usage:

  Export a single executable program as a WebSocket server:
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
