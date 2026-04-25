# HTTP & Routing

Tests for URL routing, static file serving, CGI execution, and request handling.

---

## HTTP-001: WebSocket Upgrade Request

**Priority**: P0

**Steps**:
1. Send an HTTP request with WebSocket upgrade headers:
   ```
   GET / HTTP/1.1
   Host: localhost:8080
   Upgrade: websocket
   Connection: Upgrade
   Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
   Sec-WebSocket-Version: 13
   ```
2. Observe response

**Expected Result**: HTTP 101 Switching Protocols. WebSocket connection established.

---

## HTTP-002: Plain HTTP GET (No Upgrade Headers)

**Priority**: P1
**Preconditions**: websocketd running with `cat` (no --staticdir, no --devconsole)

**Steps**:
1. `curl http://localhost:8080/`

**Expected Result**: Does NOT execute the script for plain HTTP requests. Returns an error or empty response.

---

## HTTP-003: Static File Serving - Basic

**Priority**: P1
**Preconditions**: Create static files:
```
static/index.html
static/style.css
static/app.js
static/images/logo.png
```

**Steps**:
1. `websocketd --port=8080 --staticdir=./static cat`
2. `curl http://localhost:8080/index.html` — HTML
3. `curl http://localhost:8080/style.css` — CSS
4. `curl http://localhost:8080/images/logo.png` — image

**Expected Result**: Files served with correct MIME types. Content matches the file on disk.

---

## HTTP-004: Static File Serving - Subdirectories

**Priority**: P1

**Steps**:
1. Create `static/sub/deep/page.html`
2. `curl http://localhost:8080/sub/deep/page.html`

**Expected Result**: Nested files are served correctly.

---

## HTTP-005: Static File - 404 for Missing Files

**Priority**: P1

**Steps**:
1. `curl -o /dev/null -w "%{http_code}" http://localhost:8080/nonexistent.html`

**Expected Result**: Returns HTTP 404.

---

## HTTP-006: Static File - Path Traversal Prevention

**Priority**: P0 (Security)

**Steps**:
1. `curl http://localhost:8080/../../../etc/passwd`
2. `curl http://localhost:8080/..%2F..%2F..%2Fetc%2Fpasswd`
3. `curl http://localhost:8080/%2e%2e/%2e%2e/etc/passwd`

**Expected Result**: All attempts blocked (404 or 403). No file outside static directory is served.

---

## HTTP-007: Static Files and WebSocket Coexistence

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 --staticdir=./static cat`
2. `curl http://localhost:8080/index.html` — serves file
3. Connect `ws://localhost:8080/` — WebSocket works
4. Alternate between HTTP and WebSocket requests

**Expected Result**: Both HTTP file serving and WebSocket connections work simultaneously.

---

## HTTP-008: CGI Script Execution

**Priority**: P1
**Preconditions**: Create CGI script:
```bash
#!/bin/bash
echo "Content-Type: text/plain"
echo ""
echo "Hello from CGI"
```

**Steps**:
1. `websocketd --port=8080 --cgidir=./cgi-bin cat`
2. `curl http://localhost:8080/cgi-bin/hello.cgi`

**Expected Result**: Response body "Hello from CGI" with Content-Type text/plain.

---

## HTTP-009: CGI Script with Query String

**Priority**: P1
**Preconditions**: CGI script that reads QUERY_STRING

**Steps**:
1. `curl "http://localhost:8080/cgi-bin/query.cgi?foo=bar&baz=1"`

**Expected Result**: Script receives QUERY_STRING=foo=bar&baz=1.

---

## HTTP-010: CGI Subfolder Navigation

**Priority**: P1
**Preconditions**: Create `cgi-bin/admin/status.cgi`

**Steps**:
1. `curl http://localhost:8080/cgi-bin/admin/status.cgi`

**Expected Result**: Script in subfolder executes correctly.

**Notes**: Issue #453 — cgi-dir not working in subfolders.

---

## HTTP-011: Script Directory Mode - URL Mapping

**Priority**: P0
**Preconditions**: Create script directory:
```
scripts/echo.sh (executable)
scripts/count.sh (executable)
```

**Steps**:
1. `websocketd --port=8080 --dir=scripts/`
2. Connect `ws://localhost:8080/echo.sh` — runs echo.sh
3. Connect `ws://localhost:8080/count.sh` — runs count.sh

**Expected Result**: Each URL maps to the correct script.

---

## HTTP-012: Script Directory Mode - 404 for Missing Scripts

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 --dir=scripts/`
2. Connect `ws://localhost:8080/nonexistent.sh`

**Expected Result**: Returns 404. Does not crash.

---

## HTTP-013: Script Directory Mode - Path Traversal

**Priority**: P0 (Security)

**Steps**:
1. Connect to `ws://localhost:8080/../../etc/passwd`
2. Connect to `ws://localhost:8080/%2e%2e/secret.sh`

**Expected Result**: Cannot escape the script directory. Returns 404.

---

## HTTP-014: Script Directory Mode - Non-Executable File

**Priority**: P1

**Steps**:
1. Place a non-executable file in the script directory
2. Connect to its URL

**Expected Result**: Error response — script cannot be executed. No crash.

**Notes**: Commit 11610d0 — final checks for script existence before protocol switch.

---

## HTTP-015: Dev Console Serving

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --devconsole cat`
2. `curl http://localhost:8080/`

**Expected Result**: HTML page with embedded JavaScript for the interactive dev console.

---

## HTTP-016: Custom HTTP Headers

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --header="X-Custom: test" cat`
2. `curl -I http://localhost:8080/`

**Expected Result**: Response includes `X-Custom: test` header.

---

## HTTP-017: Multiple Custom Headers

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --header="X-A: 1" --header="X-B: 2" cat`
2. `curl -I http://localhost:8080/`

**Expected Result**: Both X-A and X-B headers are present.

---

## HTTP-018: Host Header Parsing

**Priority**: P1

**Steps**:
1. Connect with `Host: example.com:8080`
2. Connect with `Host: example.com` (no port)
3. Connect with missing Host header

**Expected Result**: SERVER_NAME and SERVER_PORT are correctly derived from Host header. Missing Host is handled gracefully.

**Notes**: Fix in commit 63bf0cb for port 80 handling.

---

## HTTP-019: Concurrent HTTP and WebSocket

**Priority**: P1
**Preconditions**: websocketd with --staticdir

**Steps**:
1. Open 10 WebSocket connections
2. While connections are open, make 100 HTTP requests for static files

**Expected Result**: Both WebSocket and HTTP requests served correctly. No interference.

---

## HTTP-020: HEAD Request

**Priority**: P2

**Steps**:
1. `curl -I http://localhost:8080/` (HEAD request)

**Expected Result**: Returns appropriate headers without body. No crash.

---

## HTTP-021: URL with Query String for WebSocket

**Priority**: P1

**Steps**:
1. Connect to `ws://localhost:8080/?key=value`
2. Verify QUERY_STRING is set in child process environment

**Expected Result**: QUERY_STRING=key=value available to the script.

---

## HTTP-022: URL with Fragment (Hash)

**Priority**: P3

**Steps**:
1. Connect to `ws://localhost:8080/#fragment`

**Expected Result**: Fragment is NOT sent to the server (per HTTP spec). Connection works normally.

---

## HTTP-023: WebSocket Connection Upgrade Header Variations

**Priority**: P2

**Steps**:
1. Connect with `Connection: keep-alive, Upgrade` (multiple values)
2. Connect with `Connection: upgrade` (lowercase)
3. Connect with `Upgrade: WebSocket` (mixed case)

**Expected Result**: All valid upgrade header variations are accepted.
