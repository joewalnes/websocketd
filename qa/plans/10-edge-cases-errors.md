# Edge Cases & Error Handling

Tests for unusual inputs, error conditions, race conditions, and unexpected states.

---

## Input Edge Cases

### EDGE-001: Extremely Long Line (Text Mode)

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 cat`
2. Connect and send a single line of 10MB of text (all on one line, no newlines)

**Expected Result**: The message is delivered to the process's stdin as one line. Echo response contains the full message. No truncation.

---

### EDGE-002: Message with Only Whitespace

**Priority**: P2

**Steps**:
1. Send messages containing:
   - Single space: " "
   - Multiple spaces: "     "
   - Tab: "\t"
   - Mixed whitespace: " \t "

**Expected Result**: Whitespace is preserved and echoed back. websocketd does not strip leading/trailing whitespace (only trailing newlines).

---

### EDGE-003: Null Bytes in Text Mode

**Priority**: P2

**Steps**:
1. In text mode, send a message containing null bytes (0x00)

**Expected Result**: Behavior is defined by the text processing layer. Null bytes may cause issues with line-based processing. Document the behavior.

---

### EDGE-004: Null Bytes in Binary Mode

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --binary cat`
2. Send binary data containing null bytes

**Expected Result**: Null bytes are preserved and echoed back correctly.

---

### EDGE-005: Embedded Newlines in WebSocket Frame

**Priority**: P1

**Steps**:
1. In text mode, send a single WebSocket frame containing "line1\nline2\nline3"

**Expected Result**: The process receives three separate lines on stdin. In text mode, newlines in the WebSocket frame become line separators for the process.

---

### EDGE-006: Very Rapid Message Sending

**Priority**: P1

**Steps**:
1. Send 10,000 messages as fast as possible
2. Collect all responses

**Expected Result**: All messages are echoed back. None are lost or duplicated. Order is preserved.

---

### EDGE-007: Unicode Boundary Characters

**Priority**: P2

**Steps**:
1. Send messages containing:
   - BOM (U+FEFF)
   - Zero-width space (U+200B)
   - Right-to-left override (U+202E)
   - Surrogate pair characters (emoji)

**Expected Result**: All characters are passed through correctly. websocketd does not interpret or modify them.

---

## Error Conditions

### EDGE-008: Process Writes to Closed WebSocket

**Priority**: P1
**Preconditions**: Script that outputs continuously:
```bash
#!/bin/bash
while true; do echo "output"; sleep 0.1; done
```

**Steps**:
1. Connect to websocketd
2. Disconnect immediately (before process finishes)
3. Observe websocketd behavior

**Expected Result**: websocketd detects the closed WebSocket and terminates the process. No error loop or resource leak.

---

### EDGE-009: WebSocket Sends to Exited Process

**Priority**: P1
**Preconditions**: Script that exits after 1 second:
```bash
#!/bin/bash
sleep 1
exit 0
```

**Steps**:
1. Connect to websocketd
2. Wait 2 seconds (process has exited)
3. Try to send a message

**Expected Result**: Message is either silently discarded or the connection is closed. No crash.

---

### EDGE-010: Broken Pipe on Process STDIN

**Priority**: P1

**Steps**:
1. Start a script that closes its own stdin early:
   ```bash
   #!/bin/bash
   exec 0<&-  # close stdin
   echo "stdin closed"
   sleep 5
   ```
2. Connect and try to send messages

**Expected Result**: websocketd handles the broken pipe on stdin. No crash. Connection may be closed or messages may be lost.

---

### EDGE-011: Process That Never Reads STDIN

**Priority**: P1
**Preconditions**: Script that ignores stdin:
```bash
#!/bin/bash
echo "hello"
sleep 60
```

**Steps**:
1. Connect and send many messages
2. Observe if websocketd blocks or the buffer fills

**Expected Result**: Messages are buffered (pipe buffer). When the buffer is full, send operations may block. websocketd should handle this without crashing.

---

### EDGE-012: Disk Full During Logging

**Priority**: P3

**Steps**:
1. Start websocketd with logging to a partition that fills up
2. Connect and interact

**Expected Result**: websocketd continues functioning even if log writes fail. Connection handling is not affected.

---

### EDGE-013: Very Long Process Startup Time

**Priority**: P2
**Preconditions**: Script that takes 30 seconds to start:
```bash
#!/bin/bash
sleep 30
echo "finally ready"
while IFS= read -r line; do echo "$line"; done
```

**Steps**:
1. Connect to websocketd
2. Wait for the process to start

**Expected Result**: WebSocket connection stays open during the 30-second startup. Client receives "finally ready" eventually. No timeout from websocketd itself.

**Notes**: Issue #448 — long connection setup time concerns.

---

### EDGE-014: Simultaneous Connect and Disconnect

**Priority**: P2

**Steps**:
1. Rapidly open and close connections in parallel (50+ concurrent operations)
2. Monitor websocketd for panics or deadlocks

**Expected Result**: No panics. No goroutine leaks. No deadlocks. websocketd remains responsive.

---

## Race Conditions

### EDGE-015: Process Exit During WebSocket Send

**Priority**: P1

**Steps**:
1. Have a script that exits at a random time
2. Send messages continuously
3. Repeat many times

**Expected Result**: No panic or crash regardless of timing. Connection is closed cleanly.

---

### EDGE-016: Concurrent WebSocket Close and Process Output

**Priority**: P1

**Steps**:
1. Script outputs data rapidly
2. Client disconnects while output is being sent

**Expected Result**: No panic. No goroutine leak. Process is terminated cleanly.

---

### EDGE-017: Multiple Rapid Reconnections to Same URL

**Priority**: P1

**Steps**:
1. Connect, disconnect, reconnect rapidly 100 times to the same URL
2. Monitor for resource leaks (goroutines, file descriptors, processes)

**Expected Result**: Each cycle is independent. No resource accumulation. All processes are cleaned up.

---

## Nil/Missing Data

### EDGE-018: Missing Host Header

**Priority**: P2

**Steps**:
1. Send a WebSocket upgrade request without a Host header

**Expected Result**: Request is rejected (HTTP/1.1 requires Host). websocketd does not crash.

---

### EDGE-019: Empty Origin Header

**Priority**: P2
**Preconditions**: websocketd with --sameorigin

**Steps**:
1. Send upgrade request with `Origin:` (present but empty)

**Expected Result**: Treated as no origin or empty origin. Does not crash. Behavior is documented.

---

### EDGE-020: Malformed WebSocket Frames

**Priority**: P1

**Steps**:
1. Send raw TCP data that is not valid WebSocket framing
2. Send a frame with wrong masking
3. Send a frame with impossible length

**Expected Result**: Gorilla WebSocket library rejects invalid frames. Connection is closed. websocketd does not crash.

---

## Resource Limits

### EDGE-021: File Descriptor Exhaustion

**Priority**: P2

**Steps**:
1. Set a low file descriptor limit: `ulimit -n 64`
2. Start websocketd
3. Open many connections until file descriptors run out

**Expected Result**: New connections fail gracefully. Existing connections are not affected. websocketd does not crash.

---

### EDGE-022: Memory Pressure

**Priority**: P2

**Steps**:
1. Run websocketd under memory pressure
2. Open connections with large message payloads

**Expected Result**: Go's garbage collector handles memory. OOM killer may terminate websocketd, but it should not panic.

---

### EDGE-023: SIGPIPE Handling

**Priority**: P2

**Steps**:
1. Start websocketd
2. Have the child process output data
3. Kill websocketd (SIGTERM)
4. Observe if SIGPIPE is handled correctly

**Expected Result**: Go ignores SIGPIPE by default. No crash from broken pipes.

---

## Regression Scenarios

### EDGE-024: Nil Pointer Dereference

**Priority**: P0

**Steps**:
1. Start websocketd with `--ssl --port=8443` but no `--address`
2. Observe behavior

**Expected Result**: No panic. Either works with default address or shows clear error.

**Notes**: Regression from issue #431, commit 334a9ec. Also issue #342 — runtime nil pointer dereference.

---

### EDGE-025: Binary Frame Doubling

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 --binary cat`
2. Send a binary frame of known size (e.g., 100 bytes)
3. Verify response is exactly 100 bytes (not 200)

**Expected Result**: Response size matches input exactly.

**Notes**: Regression from commit eee5350 — slice append bug caused frames to double.

---

### EDGE-026: Process Hang After Client Disconnect

**Priority**: P0

**Steps**:
1. Start a long-running script
2. Connect, then disconnect
3. Wait 10 seconds
4. Check for zombie/orphan processes: `ps aux | grep <script>`

**Expected Result**: Process is terminated. No zombie processes.

**Notes**: Regression from issue #159, commit 3f89f2e.
