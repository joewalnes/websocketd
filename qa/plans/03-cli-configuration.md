# CLI & Configuration

Tests for command-line flag parsing, argument validation, defaults, and startup behavior.

---

## CLI-001: Default Port

**Priority**: P0

**Steps**:
1. Run `websocketd cat` (no --port)
2. Note the port in log output

**Expected Result**: Server starts on port 80 (HTTP default) or port 443 (if --ssl). Log output shows the listening address and port.

---

## CLI-002: Custom Port

**Priority**: P0

**Steps**:
1. Run `websocketd --port=9090 cat`
2. Connect to `ws://localhost:9090/`

**Expected Result**: Server starts on port 9090. Connection succeeds.

---

## CLI-003: Port Already in Use

**Priority**: P1

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. In another terminal: `websocketd --port=8080 cat`

**Expected Result**: Second instance fails with a clear error about the port being in use. First instance is unaffected.

---

## CLI-004: Invalid Port Number

**Priority**: P2

**Steps**:
1. `websocketd --port=99999 cat`
2. `websocketd --port=-1 cat`
3. `websocketd --port=abc cat`

**Expected Result**: Each case produces a clear error message. websocketd does not start.

---

## CLI-005: --address Binding (Localhost Only)

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --address=127.0.0.1 cat`
2. Connect from localhost (should succeed)
3. Connect from another machine on the LAN (should fail)

**Expected Result**: Server only listens on 127.0.0.1. External connections are refused.

---

## CLI-006: --address=0.0.0.0 (All Interfaces)

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --address=0.0.0.0 cat`
2. Connect from localhost (should succeed)
3. Connect from another LAN machine (should succeed)

**Expected Result**: Server is accessible on all network interfaces.

---

## CLI-007: Multiple --address Flags

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --address=127.0.0.1 --address=192.168.1.x cat`
2. Connect to each address

**Expected Result**: Server listens on both specified addresses. Connections succeed on both.

---

## CLI-008: IPv6 Address

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --address=[::1] cat`
2. Connect to `ws://[::1]:8080/`

**Expected Result**: Server starts on IPv6 loopback. Connection succeeds.

---

## CLI-009: Log Levels

**Priority**: P1

**Steps**: Test each level: debug, trace, access, info, error, fatal
1. Start websocketd with `--loglevel=debug`, make a connection, observe output
2. Restart with `--loglevel=error`, make a connection, observe output

**Expected Result**:
- `debug`: verbose internal state
- `trace`: per-request detail
- `access`: connection/disconnection events
- `info`: startup and important events
- `error`: only errors
- `fatal`: only fatal errors

Each level includes all levels above it in severity.

---

## CLI-010: --version Flag

**Priority**: P0

**Steps**:
1. Run `websocketd --version`

**Expected Result**: Prints version string (e.g., "websocketd 0.4.1") and exits immediately.

---

## CLI-011: --help Flag

**Priority**: P0

**Steps**:
1. Run `websocketd --help`

**Expected Result**: Prints full help text listing all flags with descriptions. Exits immediately.

**Notes**: Fix in commit 169ecec (#436).

---

## CLI-012: --license Flag

**Priority**: P2

**Steps**:
1. Run `websocketd --license`

**Expected Result**: Prints BSD-2-Clause license text. Exits immediately.

---

## CLI-013: No Command and No --dir

**Priority**: P0

**Steps**:
1. Run `websocketd --port=8080` (no command, no --dir)

**Expected Result**: Error message indicating a command or --dir is required. Help text may be shown.

---

## CLI-014: --dir and Command Together

**Priority**: P1

**Steps**:
1. Run `websocketd --port=8080 --dir=scripts/ cat`

**Expected Result**: Error: cannot specify both --dir and a command. Does not start.

---

## CLI-015: --passenv Single Variable

**Priority**: P1
**Preconditions**: Set `MY_VAR=hello` in environment

**Steps**:
1. `MY_VAR=hello websocketd --port=8080 --passenv=MY_VAR env`
2. Connect and check output

**Expected Result**: MY_VAR=hello appears in child environment. Other parent environment variables are filtered out.

---

## CLI-016: --passenv Multiple Variables

**Priority**: P1
**Preconditions**: Set `VAR1=a`, `VAR2=b`, `VAR3=c`

**Steps**:
1. `websocketd --port=8080 --passenv=VAR1,VAR2,VAR3 env`
2. Connect and check output

**Expected Result**: All three variables are available to the child process. Unspecified variables are filtered.

---

## CLI-017: --passenv Nonexistent Variable

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --passenv=DOES_NOT_EXIST env`
2. Connect and check output

**Expected Result**: No crash. DOES_NOT_EXIST is simply not present in child environment.

---

## CLI-018: --reverselookup=true

**Priority**: P2
**Preconditions**: DNS resolution available

**Steps**:
1. `websocketd --port=8080 --reverselookup env`
2. Connect and check REMOTE_HOST

**Expected Result**: REMOTE_HOST contains a DNS hostname (e.g., "localhost") rather than IP address.

---

## CLI-019: --header Custom HTTP Headers

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --header="X-Custom: value" cat`
2. Make HTTP request: `curl -I http://localhost:8080/`
3. Make WebSocket connection, inspect upgrade response headers

**Expected Result**: `X-Custom: value` appears in ALL responses (both HTTP and WebSocket upgrade).

---

## CLI-020: --header-ws vs --header-http

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --header-ws="X-WS: 1" --header-http="X-HTTP: 1" cat`
2. `curl -I http://localhost:8080/` — check headers
3. Connect via WebSocket — check upgrade response headers

**Expected Result**: HTTP response has X-HTTP but NOT X-WS. WebSocket upgrade response has X-WS but NOT X-HTTP.

---

## CLI-021: --sameorigin Flag

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --sameorigin cat`
2. Connect from same origin (page served from localhost:8080)
3. Connect from cross origin (page served from different host)

**Expected Result**: Same origin succeeds. Cross origin is rejected. See security tests for details.

---

## CLI-022: --origin Whitelist

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --origin=example.com:8080 cat`
2. Connect with `Origin: http://example.com:8080` (succeeds)
3. Connect with `Origin: http://other.com` (rejected)

**Expected Result**: Only whitelisted origin is accepted.

---

## CLI-023: --devconsole Flag

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --devconsole cat`
2. Open `http://localhost:8080/` in a browser

**Expected Result**: Interactive developer console HTML page is displayed instead of a 404/upgrade error.

---

## CLI-024: --staticdir Flag

**Priority**: P1
**Preconditions**: Create `static/index.html` with some content

**Steps**:
1. `websocketd --port=8080 --staticdir=./static cat`
2. `curl http://localhost:8080/index.html` — serves file
3. Connect `ws://localhost:8080/` — WebSocket works

**Expected Result**: HTTP requests serve static files. WebSocket connections still handled by the command.

---

## CLI-025: --cgidir Flag

**Priority**: P1
**Preconditions**: Create `cgi-bin/test.cgi` (executable, outputs HTTP headers + body)

**Steps**:
1. `websocketd --port=8080 --cgidir=./cgi-bin cat`
2. `curl http://localhost:8080/cgi-bin/test.cgi`

**Expected Result**: CGI script is executed. Output returned as HTTP response.

---

## CLI-026: --ssl Without Certificate Files

**Priority**: P1

**Steps**:
1. `websocketd --port=8443 --ssl cat` (no --sslcert, no --sslkey)

**Expected Result**: Clear error about missing certificate/key files. Does not start. No panic.

**Notes**: Panic was fixed in commit 334a9ec (issue #431).

---

## CLI-027: --ssl With Valid Cert

**Priority**: P0
**Preconditions**: Generate self-signed cert:
```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 1 -nodes -subj "/CN=localhost"
```

**Steps**:
1. `websocketd --port=8443 --ssl --sslcert=cert.pem --sslkey=key.pem cat`
2. Connect to `wss://localhost:8443/` (accepting self-signed cert)

**Expected Result**: TLS handshake succeeds. WebSocket communication works over TLS.

---

## CLI-028: --redirport HTTP Redirect

**Priority**: P1
**Preconditions**: websocketd running with SSL

**Steps**:
1. `websocketd --port=8443 --ssl --sslcert=cert.pem --sslkey=key.pem --redirport=8080 cat`
2. `curl -v http://localhost:8080/`

**Expected Result**: HTTP 301/302 redirect to `https://localhost:8443/`.

---

## CLI-029: Flag Parsing Edge Cases

**Priority**: P2

**Steps**:
1. `websocketd --port 8080 cat` (space instead of =)
2. `websocketd --unknown-flag cat`
3. `websocketd --port= cat` (empty value)

**Expected Result**: Go flag package handles these. Invalid flags produce error messages.

---

## CLI-030: --ssl With --address Not Provided

**Priority**: P1

**Steps**:
1. `websocketd --port=8443 --ssl --sslcert=cert.pem --sslkey=key.pem cat`
   (no --address flag)

**Expected Result**: Server starts normally on all interfaces. No panic.

**Notes**: Regression - panic was fixed in commit 334a9ec (issue #431).

---

## CLI-031: --binary Flag Variations

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --binary cat`
2. `websocketd --port=8080 --binary=true cat`
3. `websocketd --port=8080 --binary=false cat`

**Expected Result**: --binary and --binary=true enable binary mode. --binary=false uses text mode.
