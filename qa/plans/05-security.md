# Security

Tests for origin validation, TLS/SSL, environment variable isolation, and attack resistance.

---

## SEC-001: Same-Origin Policy - Same Origin

**Priority**: P0
**Preconditions**: `websocketd --port=8080 --sameorigin cat`

**Steps**:
1. Serve a web page from `http://localhost:8080` (using --staticdir)
2. From that page, open a WebSocket to `ws://localhost:8080/`

**Expected Result**: Connection succeeds. Origin matches.

---

## SEC-002: Same-Origin Policy - Cross Origin

**Priority**: P0
**Preconditions**: `websocketd --port=8080 --sameorigin cat`

**Steps**:
1. From a page at `http://evil.com`, try to open a WebSocket to `ws://localhost:8080/`
2. Set Origin header to `http://evil.com`

**Expected Result**: Connection rejected (HTTP 403 or equivalent).

---

## SEC-003: Same-Origin Policy - No Origin Header

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --sameorigin cat`

**Steps**:
1. Connect using a CLI tool (wscat) that does not send an Origin header

**Expected Result**: Behavior should be documented. CLI tools without Origin may be allowed or rejected depending on implementation.

---

## SEC-004: Origin Whitelist - Accepted Origin

**Priority**: P0
**Preconditions**: `websocketd --port=8080 --origin=trusted.com cat`

**Steps**:
1. Connect with `Origin: http://trusted.com`

**Expected Result**: Connection succeeds.

---

## SEC-005: Origin Whitelist - Rejected Origin

**Priority**: P0
**Preconditions**: `websocketd --port=8080 --origin=trusted.com cat`

**Steps**:
1. Connect with `Origin: http://untrusted.com`

**Expected Result**: Connection rejected.

---

## SEC-006: Origin Whitelist - Port Matching

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --origin=trusted.com:3000 cat`

**Steps**:
1. Connect with `Origin: http://trusted.com:3000` — should succeed
2. Connect with `Origin: http://trusted.com:4000` — should fail
3. Connect with `Origin: http://trusted.com` (no port) — should fail

**Expected Result**: Only exact host:port combination is accepted.

---

## SEC-007: Multiple Allowed Origins

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --origin=a.com,b.com cat`

**Steps**:
1. Connect with `Origin: http://a.com` — succeeds
2. Connect with `Origin: http://b.com` — succeeds
3. Connect with `Origin: http://c.com` — rejected

**Expected Result**: Both listed origins accepted. All others rejected.

---

## SEC-008: Null Origin

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --sameorigin cat`

**Steps**:
1. Connect with `Origin: null` (sent by sandboxed iframes, etc.)

**Expected Result**: Connection rejected. Null origin should NOT match same-origin.

**Notes**: Fixed in v0.2.10.

---

## SEC-009: file:// Origin

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --sameorigin cat`

**Steps**:
1. Open an HTML file locally (file:// URL) that tries to connect
2. Check what Origin header the browser sends

**Expected Result**: Chrome sends `Origin: null` for file:// URLs. Connection behavior depends on configuration.

**Notes**: Issues #65, #75. Commit cab80a5 addressed this.

---

## SEC-010: No Origin Restrictions (Default)

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 cat` (no --sameorigin, no --origin)
2. Connect from any origin

**Expected Result**: All connections accepted regardless of origin. This is the default behavior.

---

## SEC-011: Environment Variable Isolation

**Priority**: P0
**Preconditions**: Set sensitive vars: `export SECRET_KEY=s3cr3t`, `export DATABASE_URL=postgres://...`

**Steps**:
1. `websocketd --port=8080 env` (no --passenv)
2. Connect and examine output

**Expected Result**: SECRET_KEY and DATABASE_URL are NOT visible. Only CGI-standard variables (SERVER_SOFTWARE, REMOTE_ADDR, etc.) and the script's own environment are present.

---

## SEC-012: --passenv Selective Passing

**Priority**: P1
**Preconditions**: `export SAFE=ok UNSAFE=hidden`

**Steps**:
1. `websocketd --port=8080 --passenv=SAFE env`
2. Connect and check output

**Expected Result**: SAFE=ok present. UNSAFE NOT present.

---

## SEC-013: TLS Certificate

**Priority**: P0
**Preconditions**: Self-signed cert for localhost

**Steps**:
1. `websocketd --port=8443 --ssl --sslcert=cert.pem --sslkey=key.pem cat`
2. `openssl s_client -connect localhost:8443` — verify cert
3. Connect via `wss://localhost:8443/`

**Expected Result**: TLS handshake succeeds. Certificate details match.

---

## SEC-014: TLS Protocol Versions

**Priority**: P1
**Preconditions**: websocketd with SSL

**Steps**:
1. `openssl s_client -tls1 -connect localhost:8443` — TLS 1.0
2. `openssl s_client -tls1_1 -connect localhost:8443` — TLS 1.1
3. `openssl s_client -tls1_2 -connect localhost:8443` — TLS 1.2
4. `openssl s_client -tls1_3 -connect localhost:8443` — TLS 1.3

**Expected Result**: SSL3 is NOT supported (removed in v0.2.12). TLS 1.2 and 1.3 should work. Older versions depend on Go's TLS defaults (Go 1.18+ disables TLS 1.0/1.1 by default).

---

## SEC-015: TLS with Mismatched Cert/Key

**Priority**: P1

**Steps**:
1. `websocketd --ssl --sslcert=cert.pem --sslkey=different_key.pem cat`

**Expected Result**: Clear error about cert/key mismatch. Does not start. No panic.

---

## SEC-016: TLS with Missing Files

**Priority**: P1

**Steps**:
1. `websocketd --ssl --sslcert=/nonexistent --sslkey=/nonexistent cat`

**Expected Result**: Clear error. Does not start.

---

## SEC-017: Command Injection via URL Path

**Priority**: P0 (Security)
**Preconditions**: Script directory mode

**Steps**:
1. Connect to `ws://localhost:8080/;ls`
2. Connect to `ws://localhost:8080/$(whoami)`
3. Connect to `ws://localhost:8080/|cat%20/etc/passwd`

**Expected Result**: No shell command injection. URLs treated as file paths. Returns 404 for nonexistent scripts.

---

## SEC-018: Command Injection via Query String

**Priority**: P0 (Security)

**Steps**:
1. Connect to `ws://localhost:8080/?$(rm -rf /)`
2. Connect to `ws://localhost:8080/?;id`

**Expected Result**: Query string is passed as QUERY_STRING environment variable value. No shell interpretation.

---

## SEC-019: Header Injection

**Priority**: P1

**Steps**:
1. Send request with header containing newlines: `X-Injected: value\r\nX-Evil: injected`

**Expected Result**: Newlines in headers are sanitized or rejected. No HTTP response splitting.

---

## SEC-020: DoS - Fork Limit

**Priority**: P1
**Preconditions**: `websocketd --port=8080 --maxforks=10 cat`

**Steps**:
1. Open 10 connections (fill limit)
2. Rapidly attempt 100 more connections
3. Monitor websocketd responsiveness

**Expected Result**: Excess connections get 429 errors. websocketd does not crash. Existing connections unaffected.

---

## SEC-021: DoS - Large Headers

**Priority**: P2

**Steps**:
1. Send HTTP request with 1MB+ of custom headers

**Expected Result**: Request rejected by Go's HTTP server limits. websocketd does not crash or OOM.

---

## SEC-022: DoS - Slowloris

**Priority**: P2

**Steps**:
1. Open many TCP connections, send headers very slowly
2. Monitor websocketd resource usage

**Expected Result**: Go's HTTP server has built-in timeout handling. websocketd remains responsive.

---

## SEC-023: HTTPS Redirect Open Redirect

**Priority**: P2
**Preconditions**: websocketd with --ssl and --redirport

**Steps**:
1. `curl -H "Host: evil.com" http://localhost:8080/`

**Expected Result**: Redirect URL uses the configured host, NOT the Host header. No open redirect.

---

## SEC-024: Gorilla WebSocket Dependency

**Priority**: P0

**Steps**:
1. Check gorilla/websocket version: `go list -m github.com/gorilla/websocket`
2. Check for CVEs against that version
3. Run `govulncheck ./...` if available

**Expected Result**: No unpatched vulnerabilities. Document any known issues.

**Notes**: Issue #441 — gorilla/websocket vulnerability concern.

---

## SEC-025: Gosec Static Analysis

**Priority**: P1

**Steps**:
1. Install gosec: `go install github.com/securego/gosec/v2/cmd/gosec@latest`
2. Run: `gosec ./...`
3. Review findings

**Expected Result**: No critical/high findings. Document medium/low findings.

**Notes**: Issue #418 — Gosec SAST scan results.

---

## SEC-026: WebSocket Frame Size Limits

**Priority**: P2

**Steps**:
1. Send a WebSocket frame larger than the default gorilla/websocket read limit
2. Observe behavior

**Expected Result**: Connection is closed by the library if frame exceeds max size. No crash. No memory exhaustion.

**Notes**: Issue #445 — framesize configuration request.
