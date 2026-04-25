# Environment Variables & CGI

Tests for RFC 3875 CGI compliance, environment variable handling, and HTTP header passthrough.

---

## ENV-001: All Standard CGI Variables Present

**Priority**: P0
**Preconditions**: websocketd running with `env` command

**Steps**:
1. `websocketd --port=8080 env`
2. Connect with wscat, observe output

**Expected Result**: All of these variables are present:
- `SERVER_SOFTWARE` — "websocketd/VERSION"
- `GATEWAY_INTERFACE` — "CGI/1.1"
- `SERVER_PROTOCOL` — "HTTP/1.1"
- `SERVER_NAME` — hostname from Host header
- `SERVER_PORT` — port number
- `REQUEST_METHOD` — "GET"
- `SCRIPT_NAME` — path portion
- `REMOTE_ADDR` — client IP address
- `REMOTE_HOST` — hostname or IP

---

## ENV-002: QUERY_STRING with Parameters

**Priority**: P0

**Steps**:
1. Connect to `ws://localhost:8080/?key=value&foo=bar`
2. Check QUERY_STRING in output

**Expected Result**: `QUERY_STRING=key=value&foo=bar`

---

## ENV-003: QUERY_STRING Empty

**Priority**: P1

**Steps**:
1. Connect to `ws://localhost:8080/` (no query string)

**Expected Result**: `QUERY_STRING=` (set but empty)

---

## ENV-004: PATH_INFO and SCRIPT_NAME

**Priority**: P1
**Preconditions**: Script directory mode with echo.sh

**Steps**:
1. Connect to `ws://localhost:8080/echo.sh/extra/path`

**Expected Result**:
- `SCRIPT_NAME=/echo.sh`
- `PATH_INFO=/extra/path`

---

## ENV-005: REMOTE_ADDR and REMOTE_PORT

**Priority**: P0

**Steps**:
1. Connect to websocketd
2. Check REMOTE_ADDR and REMOTE_PORT

**Expected Result**: REMOTE_ADDR is a valid IP address (e.g., 127.0.0.1). REMOTE_PORT is a valid port number.

---

## ENV-006: UNIQUE_ID Uniqueness

**Priority**: P1

**Steps**:
1. Connect client A, note UNIQUE_ID
2. Connect client B, note UNIQUE_ID
3. Verify they differ

**Expected Result**: Each connection gets a unique UNIQUE_ID. No duplicates.

---

## ENV-007: REQUEST_URI

**Priority**: P1

**Steps**:
1. Connect to `ws://localhost:8080/path?query=1`

**Expected Result**: `REQUEST_URI=/path?query=1`

---

## ENV-008: HTTP Headers to HTTP_* Variables

**Priority**: P0

**Steps**:
1. Connect with custom headers:
   - `X-Custom-Header: custom-value`
   - `Accept-Language: en-US`
   - `User-Agent: TestClient/1.0`
2. Check environment variables

**Expected Result**:
- `HTTP_X_CUSTOM_HEADER=custom-value`
- `HTTP_ACCEPT_LANGUAGE=en-US`
- `HTTP_USER_AGENT=TestClient/1.0`
- Conversion: uppercase, hyphens to underscores, HTTP_ prefix

---

## ENV-009: HTTPS Variable with SSL

**Priority**: P1
**Preconditions**: websocketd with --ssl

**Steps**:
1. Connect via wss://
2. Check HTTPS variable

**Expected Result**: `HTTPS=on`

---

## ENV-010: HTTPS Variable without SSL

**Priority**: P1

**Steps**:
1. Connect via ws:// (no SSL)
2. Check for HTTPS variable

**Expected Result**: HTTPS is not set or empty.

---

## ENV-011: Cleared Variables

**Priority**: P2

**Steps**:
1. Connect and check AUTH_TYPE, REMOTE_IDENT, REMOTE_USER, CONTENT_TYPE, CONTENT_LENGTH

**Expected Result**: These are set but empty (per RFC 3875 for WebSocket GET requests).

---

## ENV-012: --passenv=PATH

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --passenv=PATH env`
2. Check if PATH is available

**Expected Result**: PATH from parent environment is passed to child.

**Notes**: Issue #144 — PATH env variable.

---

## ENV-013: --passenv Variable with Special Characters

**Priority**: P2
**Preconditions**: `export SPECIAL='value with spaces and "quotes"'`

**Steps**:
1. `websocketd --port=8080 --passenv=SPECIAL env`
2. Check the variable

**Expected Result**: Value preserved exactly, including spaces and quotes.

---

## ENV-014: Many HTTP Headers Conversion

**Priority**: P2

**Steps**:
1. Connect with 50+ custom HTTP headers
2. Verify all are converted to HTTP_* variables

**Expected Result**: All headers converted. No truncation.

---

## ENV-015: Cookie Header

**Priority**: P2

**Steps**:
1. Connect with `Cookie: session=abc123; token=xyz`

**Expected Result**: `HTTP_COOKIE=session=abc123; token=xyz`

---

## ENV-016: Host Header Parsing

**Priority**: P1

**Steps**:
1. Connect with `Host: example.com:8080`
2. Connect with `Host: example.com` (no port)

**Expected Result**: SERVER_NAME and SERVER_PORT correctly derived. Default port used when omitted.

**Notes**: Tested in http_test.go (tellHostPort).

---

## ENV-017: IPv6 REMOTE_ADDR

**Priority**: P2

**Steps**:
1. Connect from an IPv6 address (e.g., `::1`)
2. Check REMOTE_ADDR

**Expected Result**: REMOTE_ADDR contains the IPv6 address. SERVER_NAME reflects IPv6 if used.

---

## ENV-018: CGI Variables for CGI-Dir Scripts

**Priority**: P1
**Preconditions**: websocketd with --cgidir

**Steps**:
1. Make HTTP GET request to CGI script
2. Make HTTP POST request with body to CGI script
3. Check environment variables in each case

**Expected Result**: REQUEST_METHOD is "GET" or "POST". For POST: CONTENT_LENGTH and CONTENT_TYPE are set. Standard CGI variables are present.

---

## ENV-019: No Parent Environment Leakage

**Priority**: P0

**Steps**:
1. Set many environment variables: HOME, USER, SHELL, AWS_SECRET_KEY, etc.
2. `websocketd --port=8080 env` (no --passenv)
3. Connect and check output

**Expected Result**: Only CGI-standard variables appear. No parent environment leakage.

---

## ENV-020: SERVER_SOFTWARE Version String

**Priority**: P2

**Steps**:
1. Connect and check SERVER_SOFTWARE

**Expected Result**: Contains "websocketd/" followed by the version number matching `websocketd --version` output.
