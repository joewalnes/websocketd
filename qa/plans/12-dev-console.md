# Developer Console

Tests for the built-in interactive testing console (--devconsole flag).

---

## DEV-001: Console Page Loads

**Priority**: P0
**Preconditions**: `websocketd --port=8080 --devconsole cat`

**Steps**:
1. Open `http://localhost:8080/` in a browser

**Expected Result**: HTML page loads with the interactive WebSocket console. Includes connect/disconnect buttons, message input, and message log area.

---

## DEV-002: Connect Button

**Priority**: P0

**Steps**:
1. Open the dev console in a browser
2. Click the "Connect" button (or press Enter in URL bar)

**Expected Result**: WebSocket connection is established. Status indicator shows connected. Log shows connection event.

---

## DEV-003: Disconnect Button

**Priority**: P0

**Steps**:
1. Connect to websocketd via dev console
2. Click the "Disconnect" button

**Expected Result**: WebSocket connection is closed. Status shows disconnected. Child process is terminated.

---

## DEV-004: Send Message

**Priority**: P0

**Steps**:
1. Connect via dev console
2. Type "hello" in the message input
3. Press Enter or click Send

**Expected Result**: Message "hello" is sent. Echo response "hello" appears in the message log. Both sent and received messages are displayed.

---

## DEV-005: Message History (Arrow Keys)

**Priority**: P1

**Steps**:
1. Connect and send several messages: "msg1", "msg2", "msg3"
2. Press Up arrow key in the message input

**Expected Result**: Previous messages appear in the input field. Up arrow cycles through history. Down arrow goes forward.

---

## DEV-006: Binary Message Display

**Priority**: P2
**Preconditions**: `websocketd --port=8080 --devconsole --binary cat`

**Steps**:
1. Connect via dev console
2. Send a binary message

**Expected Result**: Binary messages are displayed (may show as "blob" indicator). Console does not crash.

---

## DEV-007: URL Bar Integration

**Priority**: P2

**Steps**:
1. Open dev console at `http://localhost:8080/`
2. Modify the WebSocket URL in the console
3. Connect

**Expected Result**: Browser URL bar updates to reflect the WebSocket path. Navigation works.

---

## DEV-008: Tab Character Display

**Priority**: P2

**Steps**:
1. Have a script output text with tab characters
2. View in dev console

**Expected Result**: Tab characters are rendered correctly (as whitespace, not literal \t).

**Notes**: Fix in commit 0e690fb.

---

## DEV-009: Mobile Viewport

**Priority**: P2

**Steps**:
1. Open dev console on a mobile device (or mobile emulation in dev tools)
2. Verify layout and usability

**Expected Result**: Console is usable on mobile screens. Viewport meta tag is present.

**Notes**: Commit efd867b added mobile-friendly viewport.

---

## DEV-010: Console Persistence

**Priority**: P3

**Steps**:
1. Send messages in the dev console
2. Refresh the page
3. Check if message history persists

**Expected Result**: Message history may use localStorage for persistence (documented in console source code).

---

## DEV-011: Dev Console Without --devconsole Flag

**Priority**: P1

**Steps**:
1. Start websocketd WITHOUT --devconsole
2. Open `http://localhost:8080/` in browser

**Expected Result**: Dev console is NOT served. Returns appropriate error or 404.

---

## DEV-012: Dev Console with --staticdir

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --devconsole --staticdir=./static cat`
2. Open `http://localhost:8080/` in browser
3. Open `http://localhost:8080/index.html`

**Expected Result**: Document precedence. One of devconsole or staticdir takes priority for the root path.

---

## DEV-013: Multiple Browser Windows

**Priority**: P1

**Steps**:
1. Open dev console in two browser windows
2. Connect both
3. Send messages from each

**Expected Result**: Each window has an independent WebSocket connection and child process. No cross-contamination.

---

## DEV-014: Reconnect After Server Restart

**Priority**: P2

**Steps**:
1. Open dev console and connect
2. Stop websocketd
3. Restart websocketd
4. Try reconnecting from the dev console

**Expected Result**: Console detects the disconnect. Reconnection works after server restart.
