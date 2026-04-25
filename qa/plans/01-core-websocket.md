# Core WebSocket Functionality

Tests for the fundamental WebSocket connection lifecycle and messaging behavior.

---

## CORE-001: Basic WebSocket Connection

**Priority**: P0
**Preconditions**: websocketd running with a simple echo script (`cat` or equivalent)

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect with a WebSocket client: `wscat -c ws://localhost:8080/`
3. Observe the connection is established

**Expected Result**: WebSocket connection opens successfully. Server responds with HTTP 101 Switching Protocols. No errors in websocketd log output.

---

## CORE-002: Send and Receive Text Message

**Priority**: P0
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect with wscat: `wscat -c ws://localhost:8080/`
3. Type "hello world" and press Enter
4. Observe the response

**Expected Result**: Server echoes back "hello world" as a text WebSocket frame.

---

## CORE-003: Multiple Messages in Sequence

**Priority**: P0
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect with wscat
3. Send "message 1", wait for response
4. Send "message 2", wait for response
5. Send "message 3", wait for response

**Expected Result**: Each message is echoed back in order. No messages lost or reordered.

---

## CORE-004: Server-Initiated Messages (STDOUT Output)

**Priority**: P0
**Preconditions**: Create a script `count.sh`:
```bash
#!/bin/bash
for i in 1 2 3 4 5; do
  echo $i
  sleep 1
done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./count.sh`
2. Connect with wscat

**Expected Result**: Client receives "1", "2", "3", "4", "5" as separate WebSocket text frames, approximately 1 second apart. Connection closes after the script exits.

---

## CORE-005: Client Disconnect

**Priority**: P0
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect with wscat
3. Send a message, verify echo
4. Close the WebSocket connection (Ctrl+C in wscat)
5. Check websocketd logs

**Expected Result**: Connection closes cleanly. The child process (cat) is terminated. Log shows disconnect event. No zombie processes remain.

---

## CORE-006: Server-Side Process Exit

**Priority**: P0
**Preconditions**: Create a script that exits after one message:
```bash
#!/bin/bash
read line
echo "got: $line"
exit 0
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./oneshot.sh`
2. Connect with wscat
3. Send "hello"
4. Observe the response and connection state

**Expected Result**: Client receives "got: hello". WebSocket connection is closed by the server after the process exits.

---

## CORE-007: Empty Message Handling

**Priority**: P1
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Connect to websocketd
2. Send an empty string (just newline)
3. Observe behavior

**Expected Result**: An empty line is sent to the process's stdin. The process responds with an empty line (empty WebSocket text frame).

---

## CORE-008: Binary Mode - Basic

**Priority**: P1
**Preconditions**: websocketd running with `cat` in binary mode

**Steps**:
1. Start websocketd: `websocketd --port=8080 --binary cat`
2. Connect with a WebSocket client that supports binary frames
3. Send a binary message containing bytes: `0x00 0x01 0x02 0xFF`
4. Observe the response

**Expected Result**: Server echoes back the exact binary data. Frame type is binary (opcode 0x02), not text. No line-ending processing occurs.

**Notes**: Regression - binary frame doubling bug was fixed in commit eee5350 (May 2016). Verify data length matches.

---

## CORE-009: Binary Mode - Large Payload

**Priority**: P2
**Preconditions**: websocketd running with `cat` in binary mode

**Steps**:
1. Start websocketd: `websocketd --port=8080 --binary cat`
2. Connect and send a 1MB binary payload
3. Verify the echoed response matches exactly

**Expected Result**: All bytes are received correctly. No truncation or corruption.

---

## CORE-010: Text Mode - Line Buffering

**Priority**: P0
**Preconditions**: websocketd running with `cat` in default (text) mode

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect with wscat
3. Send "line one"
4. Send "line two"

**Expected Result**: Each line sent by the client becomes one line on the process's stdin. Each line written by the process to stdout becomes one WebSocket text frame. Lines are delimited by `\n`.

---

## CORE-011: Text Mode - Newline Stripping

**Priority**: P1
**Preconditions**: Create a script that outputs lines with different endings:
```bash
#!/bin/bash
printf "unix newline\n"
printf "windows newline\r\n"
printf "no newline"
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./newlines.sh`
2. Connect with wscat
3. Observe received messages

**Expected Result**: Each line becomes a separate WebSocket message. Trailing `\n` and `\r\n` are stripped from each message. "no newline" is delivered when the process exits (or as a final message).

**Notes**: trimEOL function handles both `\n` and `\r\n`. See endpoint_test.go for unit tests.

---

## CORE-012: Multiple Concurrent Connections

**Priority**: P0
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Open connection A with wscat
3. Open connection B with wscat (different terminal)
4. Send "from A" on connection A
5. Send "from B" on connection B
6. Observe responses

**Expected Result**: Connection A receives "from A". Connection B receives "from B". Each connection has its own child process. Messages do not cross between connections.

---

## CORE-013: Rapid Connect/Disconnect

**Priority**: P2
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. In a loop, connect and immediately disconnect 100 times
3. Monitor websocketd for errors, crashes, or resource leaks
4. After the loop, verify websocketd still accepts new connections

**Expected Result**: No crashes, no resource leaks, no zombie processes. websocketd remains functional.

---

## CORE-014: Large Text Message

**Priority**: P2
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect and send a single text message containing 1MB of text (one very long line)
3. Observe the response

**Expected Result**: The full message is echoed back correctly.

---

## CORE-015: Unicode Text Messages

**Priority**: P1
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Start websocketd: `websocketd --port=8080 cat`
2. Connect and send messages containing:
   - ASCII: "hello"
   - Latin extended: "cafe\u0301"
   - CJK characters: "你好世界"
   - Emoji: "🎉🚀"
   - Mixed: "hello 世界 🌍"
3. Verify each is echoed correctly

**Expected Result**: All Unicode text is preserved and echoed back correctly.

**Notes**: Issue #348 reports problems with Chinese characters. This may be a buffering or encoding issue in certain script languages.

---

## CORE-016: WebSocket Close Frame

**Priority**: P1
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Connect to websocketd
2. Send a WebSocket close frame with status code 1000 (normal closure)
3. Observe the server's response

**Expected Result**: Server acknowledges the close frame and terminates the connection and the child process cleanly.

---

## CORE-017: WebSocket Ping/Pong

**Priority**: P2
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Connect to websocketd
2. Send a WebSocket ping frame
3. Observe the response

**Expected Result**: Server responds with a pong frame. Connection remains open. This is handled by the gorilla/websocket library.

---

## CORE-018: Connection After Process Crash

**Priority**: P1
**Preconditions**: Create a script that crashes:
```bash
#!/bin/bash
echo "starting"
exit 1
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./crash.sh`
2. Connect with wscat
3. Observe behavior
4. Disconnect, then connect again

**Expected Result**: First connection receives "starting" and then the connection is closed. Second connection also works normally (new process is spawned). Exit code 1 from the process does not crash websocketd.

---

## CORE-019: Non-Existent Command

**Priority**: P1
**Preconditions**: None

**Steps**:
1. Start websocketd: `websocketd --port=8080 /nonexistent/command`
2. Attempt to connect with wscat

**Expected Result**: websocketd starts but connecting fails gracefully. The server should return an appropriate error. No panic or crash.

---

## CORE-020: Binary Mode Flag Variations

**Priority**: P2
**Preconditions**: None

**Steps**:
1. Test `--binary` (no value)
2. Test `--binary=true`
3. Test `--binary=false`
4. Test with no binary flag (default)

**Expected Result**:
- `--binary` and `--binary=true` enable binary mode
- `--binary=false` and no flag use text mode (default)
