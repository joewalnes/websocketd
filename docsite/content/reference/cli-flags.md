---
title: "CLI flags"
weight: 10
description: "Every websocketd command line flag, its default, and exactly what it does."
---

Every flag websocketd accepts, in alphabetical order, with its default and its
exact effect. Order on the command line matters: see [flag placement](/reference/).

For a bare list with no notes, run `websocketd --help`.

## Flags

### `--address`

**Default:** `none (may be given multiple times)`

Interfaces to bind to (e.g. 127.0.0.1 or [::1]).

May be given more than once to bind several interfaces. Every address listens on the same --port.

### `--anyorigin`

**Default:** `false`

Explicitly accept any origin (the current default) and silence the origin-policy startup warning.

States the current permissive-origin default explicitly and silences the origin-policy warning websocketd prints to stderr at startup. Combining it with --sameorigin or --origin is rejected at startup, since those restrict what this flag accepts. A future websocketd release defaults to --sameorigin instead. See [security model](/understanding/security-model/).

### `--binary`

**Default:** `false`

Set websocketd to experimental binary mode (default is line by line).

Switches the WebSocket message type to binary frames instead of text, and removes the newline framing rule: output is forwarded in raw chunks as read, with no newline required, and no newline is appended to input. It does not allocate a pseudo-terminal, so a program that only behaves interactively under a terminal still does not do so here (issue #443). See [message framing](/understanding/message-framing/).

### `--cgidir=CGIDIR`

**Default:** `"" (empty)`

Serve CGI scripts from this directory over HTTP.

Scripts here run as CGI programs answering ordinary HTTP requests, not WebSocket upgrades. The request path must name the script file exactly; there is no extra path information after it. When the CGI directory sits inside the --staticdir tree, its scripts also answer at their path there: with --staticdir=/PAGE --cgidir=/PAGE/cgi-bin, /cgi-bin/hello.sh runs the same script as /hello.sh. That position is decided by which directory each flag names rather than by how it is spelled, so a symlinked or differently-cased pair of paths routes the same way, and a --cgidir genuinely outside the static tree is not brought inside it by a symlink there. It is worked out once and reused, so a deployment symlink flipped under a running server does not re-route requests; restart to pick up a new layout. Files in the CGI directory are never served as static content. The variables a CGI script receives differ from the WebSocket ones, because Go's net/http/cgi builds them: see [environment variables](/reference/environment-variables/). May be combined with --dir, which serves a separate directory over WebSocket.

### `--closems=CLOSEMS`

**Default:** `0`

Extra time added to each of the first three waits before websocketd escalates to the next termination signal.

The value is added to each of the first three steps of the teardown ladder: closing stdin (100 ms), SIGINT (250 ms), and SIGTERM (500 ms). The final SIGKILL step waits a fixed 1000 ms and is not affected. Signals are sent whatever the value; 0 means no extra delay, not no signals. See [process lifecycle](/understanding/process-lifecycle/).

### `--devconsole`

**Default:** `false`

Enable development console (cannot be used in conjunction with --staticdir or --cgidir).

All three of --devconsole, --staticdir and --cgidir claim the same non-WebSocket HTTP surface, so websocketd exits with code 4 if the console is combined with either of the others. See [dev console](/reference/dev-console/).

### `--dir=DIR`

**Default:** `"" (empty)`

Base directory for WebSocket scripts.

Any file under this directory is reachable as its own WebSocket endpoint, named after its path within the directory. The matched path becomes SCRIPT_NAME and anything left over becomes PATH_INFO. The executable bit is not checked when the path is resolved; whether the file runs is decided when websocketd launches it. A file reached through a symlink that leaves the directory is answered with 404. Files here are never served as static content, so a plain GET of a script that also sits inside a --staticdir tree returns 404 rather than the script's source. A script keeps the URL --dir gives it and gains no second one from where the directory sits: unlike --cgidir, no URL prefix is derived for it inside --staticdir. Giving both --dir and a COMMAND is rejected at startup. May be combined with --cgidir, which serves a separate directory as CGI over HTTP.

### `--header`

**Default:** `none (may be given multiple times)`

Custom headers for any response.

May be given more than once. Added to successful WebSocket upgrade responses and to every response the WebSocket handler did not produce: the dev console, CGI, static files, and 404s. Error responses from the WebSocket handler itself, such as a rejected upgrade or a 429, carry no configured headers.

### `--header-http`

**Default:** `none (may be given multiple times)`

Custom headers for all but WebSocket upgrade HTTP responses.

May be given more than once. Added to every response the WebSocket handler did not produce: the dev console, CGI, static files, and 404s.

### `--header-ws`

**Default:** `none (may be given multiple times)`

Custom headers for successful WebSocket upgrade responses.

May be given more than once. Added to successful WebSocket upgrade responses only.

### `--help`

**Default:** `false`

Print help and exit.

### `--license`

**Default:** `false`

Print license and exit.

### `--loglevel=LOGLEVEL`

**Default:** `access`

Log level, one of: debug, trace, access, info, error, fatal, none.

### `--maxforks=MAXFORKS`

**Default:** `1024`

Max forks, zero means unlimited.

Each WebSocket connection and each CGI request is a full subprocess, and this caps how many may be live at once. Beyond the cap, an upgrade or CGI request is answered with 429 Too Many Requests rather than queued. Zero means unlimited. Operators running many concurrent long-lived connections have hit the default ceiling (issues #226, #228, #356). See [the process model](/understanding/process-model/).

### `--maxframesize=MAXFRAMESIZE`

**Default:** `1048576`

Max inbound WebSocket message size in bytes (0 = unlimited).

An inbound WebSocket message larger than this limit is rejected and the connection is closed with status 1009 (message too big), which bounds how much one client can make websocketd buffer. Zero disables the limit. A negative value is rejected at startup with exit code 1, because it would otherwise read as unlimited and silently remove the limit (issue #472). See [security model](/understanding/security-model/).

### `--origin=ORIGIN`

**Default:** `"" (empty)`

Restrict upgrades if origin does not match the list.

Entries are comma-separated. An entry with no scheme (just host[:port]) matches both http and https origins; an entry prefixed "https://" matches https only. An entry with an explicit port matches that port only. An entry with no port matches only the default port of a scheme it accepts, 80 for http and 443 for https. Appending ":*" to an entry matches any port on that host (issue #473). Combining this flag with --anyorigin is rejected at startup. See [security model](/understanding/security-model/).

### `--passenv=PASSENV`

**Default:** `platform-dependent`

List of envvars to pass to subprocesses (others will be cleaned out).

The default is PATH plus the platform's shared library search path: PATH,LD_LIBRARY_PATH on Linux, PATH,DYLD_LIBRARY_PATH on macOS, and PATH,SystemRoot,COMSPEC,PATHEXT,WINDIR on Windows. Passing --passenv REPLACES that default rather than adding to it, so --passenv=API_KEY alone leaves the child with no PATH. A named variable that is empty or unset in websocketd's own environment is dropped, not forwarded as an empty string. HTTPS is always skipped, because that variable is websocketd's own --ssl signal. This flag controls only which of websocketd's OWN environment variables reach the child; per-request CGI variables such as QUERY_STRING are built separately and are unaffected by it, a recurring point of confusion (issues #202, #223, #312, #391). See [passing data into your script](/how-to/patterns/pass-arguments/).

### `--passstderr`

**Default:** `false`

Forward STDERR to WebSocket clients as tagged JSON messages, alongside tagged STDOUT (mutually exclusive with --binary).

The child's stdout and stderr both reach the client as JSON objects of the form {"stream":"stdout","data":"..."}, one per line of output, so the two streams can be told apart. Plain (untagged) output is no longer sent. stderr is still written to websocketd's own log as well. Combining it with --binary is rejected at startup with exit code 1.

### `--pingms=PINGMS`

**Default:** `0`

WebSocket ping interval in milliseconds (0 disables).

Zero disables pings entirely, and an idle connection is then never timed out; this is the default, and it accounts for a long run of "mystery disconnect" reports where the connection was in fact dropped by something in between (issues #37, #209, #260, #275, #439). A non-zero value sends a WebSocket ping every interval and sets a read deadline of twice the interval. Only an incoming pong resets that deadline, so a client that keeps sending data but never answers a ping is disconnected within 2x --pingms just the same. See [process lifecycle](/understanding/process-lifecycle/).

### `--port=PORT`

**Default:** `0`

HTTP port to listen on.

The default of 0 is not a real port: it means 80, or 443 when --ssl is given. Every --address binds this same port.

### `--raw`

**Default:** `false`

Set websocketd to experimental raw mode (like text mode but char by char instead of line by line).

### `--redirport=REDIRPORT`

**Default:** `0`

HTTP port to redirect to canonical --port address.

Runs a second, plain HTTP listener on this port whose only response is a 301 redirect to the main listener. The redirect target keeps the host the client itself sent, along with the path and query it asked for, and rewrites only the scheme and the port, so a link into the site arrives at that link and it is not an open redirect.

### `--reverselookup`

**Default:** `false`

Perform reverse DNS lookups on remote clients.

Sets REMOTE_HOST to the result of a reverse DNS lookup of the client's address instead of the address itself. The lookup happens on every connection. It has no effect on --cgidir requests, where net/http/cgi sets REMOTE_HOST to the client IP regardless. See [environment variables](/reference/environment-variables/).

### `--sameorigin`

**Default:** `false`

Restrict upgrades if origin and host headers differ.

Accepts an upgrade only when the host and port in the request's Origin header match the host and port in its Host header. A request with no Origin header is treated as origin "file:" and does not match. Combining it with --anyorigin is rejected at startup. See [security model](/understanding/security-model/).

### `--socketmode=SOCKETMODE`

**Default:** `"" (empty)`

Octal permission bits to force on the --unixsocket file (e.g. 0700); default follows umask.

Applied with chmod to the socket file immediately after bind, so the window in which the umask-derived mode applies is as short as it can be. An empty value leaves the mode to the process umask; websocketd does not change the default socket permissions (issue #474). A value of 0, a value above 0777, and a value that is not octal are each rejected at startup with exit code 1. It applies only to --unixsocket.

### `--ssl`

**Default:** `false`

Use TLS on listening socket (see also --sslcert and --sslkey).

Requires both --sslcert and --sslkey; giving --ssl without them, or either of them without --ssl, is rejected at startup with exit code 1. The listener negotiates TLS 1.2 or higher. The spawned process additionally receives HTTPS=on. See [serve over wss://](/how-to/deploy/tls/).

### `--sslca=SSLCA`

**Default:** `"" (empty)`

CA certificate file for client certificate verification (mutual TLS).

Turns on mutual TLS: client certificates are required and verified against this CA file. It requires --ssl; giving --sslca without --ssl is rejected at startup with exit code 1, because mutual TLS has no handshake to verify a client certificate in without a TLS listener (issue #477). See [serve over wss://](/how-to/deploy/tls/).

### `--sslcert=SSLCERT`

**Default:** `"" (empty)`

Should point to certificate PEM file when --ssl is used.

Read only when --ssl is given. Giving it without --ssl is rejected at startup with exit code 1.

### `--sslkey=SSLKEY`

**Default:** `"" (empty)`

Should point to certificate private key file when --ssl is used.

Read only when --ssl is given. Giving it without --ssl is rejected at startup with exit code 1.

### `--staticdir=STATICDIR`

**Default:** `"" (empty)`

Serve static content from this directory over HTTP.

Four kinds of request are answered with 404 rather than served: a path with a segment beginning with "." (.git, .env, .ssh, ...); a directory with no index.html file, since directories are never listed; a file reached through a symlink that leaves the directory; and any file inside a --dir or --cgidir tree, whichever URL reaches it. That last refusal is dropped when --staticdir names one of those directories itself, or a directory inside one. A first path segment of exactly .well-known is exempt from the dotfile rule, so ACME challenges and security.txt are still served; a dotfile nested deeper inside it is not. None of that vets the directory's contents: a secret under an ordinary name is still served, so point --staticdir at a tree holding only what is meant to be public. Rejected together with --devconsole. Why each of these is refused is in the [security model](/understanding/security-model/).

### `--unixsocket=UNIXSOCKET`

**Default:** `"" (empty)`

Path of a Unix domain socket to listen on, in addition to (or instead of) --address/--port.

Served in addition to the TCP listener. When --unixsocket is the only listening flag given, with no --port, --address, or --redirport, no TCP listener is started at all. At startup a socket file already at the path is probed: if nothing is listening on it, it is removed as stale and rebound; if something is, websocketd exits with code 3 rather than making the running server unreachable. The file is not removed on exit.

### `--version`

**Default:** `false`

Print version and exit.

Generated from websocketd's flag definitions by `tools/gendocs`. Edit the generator, not this page.
