# Protocol Compatibility

Tests for WebSocket protocol versions, HTTP versions, and browser compatibility.

---

## WebSocket Protocol

### PROTO-001: WebSocket Version 13 (RFC 6455)

**Priority**: P0

**Steps**:
1. Connect with `Sec-WebSocket-Version: 13`
2. Verify connection succeeds

**Expected Result**: Standard WebSocket protocol (version 13, RFC 6455) is fully supported.

---

### PROTO-002: Unsupported WebSocket Version

**Priority**: P2

**Steps**:
1. Send upgrade request with `Sec-WebSocket-Version: 8`
2. Observe response

**Expected Result**: Connection rejected or version negotiation occurs. Gorilla WebSocket library handles this.

---

### PROTO-003: Missing Sec-WebSocket-Key

**Priority**: P2

**Steps**:
1. Send upgrade request without `Sec-WebSocket-Key` header

**Expected Result**: Upgrade rejected. Proper error response.

---

### PROTO-004: WebSocket Close Codes

**Priority**: P1

**Steps**: Test various close codes:
1. Close with code 1000 (normal closure)
2. Close with code 1001 (going away)
3. Close with code 1006 (abnormal closure — connection dropped)
4. Close with code 1011 (unexpected condition)

**Expected Result**: Each close code is handled gracefully. Child process is terminated. websocketd logs the close reason.

**Notes**: Issue #456 — detecting ungraceful closure. Issue #399 — iOS abnormal closure 1006.

---

### PROTO-005: WebSocket Extensions

**Priority**: P2

**Steps**:
1. Connect with `Sec-WebSocket-Extensions: permessage-deflate`
2. Observe if compression is supported

**Expected Result**: Document whether permessage-deflate is supported. Gorilla WebSocket has optional compression support. Connection should not fail even if extension is not supported.

---

### PROTO-006: WebSocket Subprotocols

**Priority**: P2

**Steps**:
1. Connect with `Sec-WebSocket-Protocol: chat, superchat`
2. Observe the response

**Expected Result**: Document subprotocol handling behavior. websocketd does not implement subprotocol negotiation, so the header may be ignored.

---

### PROTO-007: Large WebSocket Frames

**Priority**: P1

**Steps**:
1. Send a single WebSocket frame of 64KB
2. Send a single WebSocket frame of 1MB
3. Send a single WebSocket frame of 10MB

**Expected Result**: Frames within the library's default limit are handled. Oversized frames may cause connection closure per the library's configuration.

**Notes**: Issue #445 — framesize configuration request.

---

### PROTO-008: Fragmented WebSocket Messages

**Priority**: P2

**Steps**:
1. Send a WebSocket message split across multiple frames (fragmentation)
2. Observe reassembly

**Expected Result**: Gorilla WebSocket library reassembles fragmented messages. The complete message is delivered to the child process.

---

### PROTO-009: WebSocket Handshake Timeout

**Priority**: P2

**Steps**:
1. Open a TCP connection to websocketd
2. Send partial upgrade headers very slowly
3. Wait and observe

**Expected Result**: Connection times out after the handshake timeout period. websocketd is not stuck waiting indefinitely.

**Notes**: Commit b2b6022 added WebSocket handshake timeout.

---

## HTTP Versions

### PROTO-010: HTTP/1.1

**Priority**: P0

**Steps**:
1. `curl --http1.1 http://localhost:8080/`
2. Connect via WebSocket over HTTP/1.1

**Expected Result**: Full functionality with HTTP/1.1. This is the standard transport for WebSocket.

---

### PROTO-011: HTTP/1.0

**Priority**: P2

**Steps**:
1. `curl --http1.0 http://localhost:8080/`

**Expected Result**: Static files may be served. WebSocket upgrade requires HTTP/1.1, so WebSocket over HTTP/1.0 should fail gracefully.

---

### PROTO-012: HTTP/2

**Priority**: P2

**Steps**:
1. `curl --http2 http://localhost:8080/`

**Expected Result**: Document behavior. Go's HTTP server supports HTTP/2 by default with TLS. WebSocket over HTTP/2 (RFC 8441) may or may not be supported by the gorilla/websocket library.

---

## Browser Compatibility

### BROWSER-001: Chrome (Latest)

**Priority**: P0
**Preconditions**: websocketd with --devconsole or --staticdir with a test HTML page

**Steps**:
1. Open Chrome, navigate to the websocketd page
2. Open WebSocket connection from JavaScript:
   ```javascript
   var ws = new WebSocket("ws://localhost:8080/");
   ws.onmessage = function(e) { console.log(e.data); };
   ws.onopen = function() { ws.send("hello"); };
   ```
3. Verify message exchange

**Expected Result**: WebSocket connection works in Chrome. Messages sent and received correctly.

---

### BROWSER-002: Firefox (Latest)

**Priority**: P0

**Steps**:
1. Same as BROWSER-001 but in Firefox

**Expected Result**: Full WebSocket functionality in Firefox.

---

### BROWSER-003: Safari (Latest)

**Priority**: P0

**Steps**:
1. Same as BROWSER-001 but in Safari on macOS

**Expected Result**: Full WebSocket functionality in Safari.

---

### BROWSER-004: Edge (Latest, Chromium-based)

**Priority**: P1

**Steps**:
1. Same as BROWSER-001 but in Edge

**Expected Result**: Full WebSocket functionality in Edge.

---

### BROWSER-005: Mobile Safari (iOS)

**Priority**: P1

**Steps**:
1. Access websocketd from iOS Safari
2. Test WebSocket connection

**Expected Result**: WebSocket works on mobile Safari.

**Notes**: Issue #399 — iOS close 1006 abnormal closure. Specifically test connection stability and close handling.

---

### BROWSER-006: Mobile Chrome (Android)

**Priority**: P1

**Steps**:
1. Access websocketd from Android Chrome
2. Test WebSocket connection

**Expected Result**: WebSocket works on mobile Chrome.

---

### BROWSER-007: Browser - wss:// (Secure WebSocket)

**Priority**: P0
**Preconditions**: websocketd with --ssl

**Steps**:
1. Open any browser
2. Connect to `wss://localhost:8443/` from JavaScript
3. Verify functionality

**Expected Result**: Secure WebSocket works in all major browsers.

---

### BROWSER-008: Browser - Mixed Content (ws:// from https://)

**Priority**: P1

**Steps**:
1. Serve a page over HTTPS
2. From that page, try to connect to `ws://localhost:8080/` (non-secure WebSocket)

**Expected Result**: Modern browsers block mixed content. Document the expected behavior (connection should fail with a security error in the browser).

---

### BROWSER-009: Browser - Dev Console UI

**Priority**: P1
**Preconditions**: websocketd with --devconsole

**Steps**:
1. Open the dev console in Chrome, Firefox, Safari, Edge
2. Connect, send messages, disconnect
3. Test message history (up/down arrow keys)
4. Test binary message display

**Expected Result**: Dev console works in all major browsers. UI is functional and responsive.

**Notes**: Commit efd867b added mobile-friendly viewport for dev console.

---

### BROWSER-010: Browser - Tab Character Display

**Priority**: P2

**Steps**:
1. Have a script output text containing tab characters
2. View in the dev console

**Expected Result**: Tab characters are displayed correctly (not as literal `\t`).

**Notes**: Fix in commit 0e690fb.

---

## Client Library Compatibility

### CLIENT-001: wscat (Node.js)

**Priority**: P0

**Steps**:
1. `npm install -g wscat`
2. `wscat -c ws://localhost:8080/`
3. Send messages, verify echo

**Expected Result**: wscat works with websocketd.

---

### CLIENT-002: Python websockets Library

**Priority**: P1

**Steps**:
1. Use Python's `websockets` library:
   ```python
   import asyncio, websockets
   async def test():
       async with websockets.connect("ws://localhost:8080/") as ws:
           await ws.send("hello")
           print(await ws.recv())
   asyncio.run(test())
   ```

**Expected Result**: Python websockets library works with websocketd.

---

### CLIENT-003: Go gorilla/websocket Client

**Priority**: P1

**Steps**:
1. Write a Go client using gorilla/websocket
2. Connect, send, receive

**Expected Result**: gorilla/websocket client works with websocketd.

---

### CLIENT-004: curl WebSocket (curl 7.86+)

**Priority**: P2

**Steps**:
1. `curl --include --no-buffer --header "Connection: Upgrade" --header "Upgrade: websocket" --header "Sec-WebSocket-Version: 13" --header "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" http://localhost:8080/`

**Expected Result**: curl shows the WebSocket upgrade response.

---

### CLIENT-005: websocat

**Priority**: P2

**Steps**:
1. `websocat ws://localhost:8080/`
2. Send messages, verify echo

**Expected Result**: websocat works with websocketd.
