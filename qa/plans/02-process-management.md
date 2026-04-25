# Process Management

Tests for child process lifecycle, stdio piping, signal handling, and resource limits.

---

## PROC-001: Each Connection Gets Its Own Process

**Priority**: P0
**Preconditions**: Create a script that prints its PID:
```bash
#!/bin/bash
echo "$$"
while IFS= read -r line; do echo "$line"; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./pid.sh`
2. Connect client A, note the PID received
3. Connect client B, note the PID received

**Expected Result**: PIDs are different. Each WebSocket connection spawns a new child process.

---

## PROC-002: STDIN from WebSocket to Process

**Priority**: P0
**Preconditions**: websocketd running with `cat`

**Steps**:
1. Connect to websocketd
2. Send "test input" via WebSocket
3. Observe the process's response

**Expected Result**: The process receives "test input\n" on its stdin and echoes it back.

---

## PROC-003: STDOUT from Process to WebSocket

**Priority**: P0
**Preconditions**: Create a script that generates output on startup:
```bash
#!/bin/bash
echo "welcome"
echo "ready"
while IFS= read -r line; do echo "$line"; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./welcome.sh`
2. Connect with wscat (do not send anything)

**Expected Result**: Client receives "welcome" and "ready" as two separate WebSocket messages immediately after connecting.

---

## PROC-004: STDERR Goes to Logs, Not WebSocket

**Priority**: P0
**Preconditions**: Create a script that writes to both stdout and stderr:
```bash
#!/bin/bash
echo "stdout message"
echo "stderr message" >&2
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 --loglevel=debug ./both.sh`
2. Connect with wscat
3. Observe client messages and server log output

**Expected Result**: Client receives only "stdout message". "stderr message" appears in the websocketd log output, NOT sent to the client via WebSocket.

---

## PROC-005: Graceful Termination Signal Sequence

**Priority**: P0
**Preconditions**: Create a script that traps signals and logs them:
```bash
#!/bin/bash
trap 'echo "GOT SIGINT" >> /tmp/ws-signals.log' INT
trap 'echo "GOT SIGTERM" >> /tmp/ws-signals.log' TERM
echo "ready"
while true; do sleep 0.1; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./trap.sh`
2. Connect, wait for "ready"
3. Disconnect the WebSocket client
4. Wait 1 second, check /tmp/ws-signals.log

**Expected Result**: Process receives escalating termination signals. Default timing: SIGINT at 100ms, SIGTERM at 250ms, SIGKILL at 500ms. Process is fully terminated.

**Notes**: Process hanging was a bug fixed in commit 3f89f2e (issue #159).

---

## PROC-006: --closems Custom Grace Period

**Priority**: P1
**Preconditions**: Same trapping script as PROC-005

**Steps**:
1. Start websocketd: `websocketd --port=8080 --closems=3000 ./trap.sh`
2. Connect, wait for "ready"
3. Disconnect and time signal delivery

**Expected Result**: Signals are delayed according to --closems. SIGINT at 3000ms, SIGTERM at 7500ms, SIGKILL at 15000ms. Process has more time to clean up.

---

## PROC-007: --closems=0 Immediate Kill

**Priority**: P2
**Steps**:
1. Start websocketd: `websocketd --port=8080 --closems=0 ./trap.sh`
2. Connect then disconnect
3. Check that the process is killed immediately

**Expected Result**: Process is terminated with minimal delay.

---

## PROC-008: --maxforks Limit Enforced

**Priority**: P0
**Preconditions**: Create a long-running script:
```bash
#!/bin/bash
echo "connected"
while true; do sleep 1; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 --maxforks=2 ./long.sh`
2. Connect client A (succeeds, receives "connected")
3. Connect client B (succeeds, receives "connected")
4. Connect client C

**Expected Result**: Client C receives HTTP 429 (Too Many Requests). The WebSocket upgrade is rejected. Clients A and B continue working normally.

**Notes**: Related to issue #366 — client cannot distinguish fork limit from server error.

---

## PROC-009: --maxforks Recovery After Disconnect

**Priority**: P1
**Preconditions**: Same as PROC-008

**Steps**:
1. Start websocketd: `websocketd --port=8080 --maxforks=1 ./long.sh`
2. Connect client A (succeeds)
3. Connect client B (fails with 429)
4. Disconnect client A
5. Wait briefly, then connect client B again

**Expected Result**: After client A disconnects, the fork counter is decremented. Client B can now connect successfully.

---

## PROC-010: Process Exit Code Non-Zero

**Priority**: P1
**Preconditions**: Create a script:
```bash
#!/bin/bash
echo "about to fail"
exit 42
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./fail.sh`
2. Connect with wscat

**Expected Result**: Client receives "about to fail". Connection is closed. Exit code 42 is logged. websocketd does not crash.

---

## PROC-011: Process Crashes (Segfault)

**Priority**: P1
**Preconditions**: Compile a C program that segfaults:
```c
#include <stdlib.h>
int main() { *(int*)0 = 0; return 0; }
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./segfault`
2. Connect with wscat

**Expected Result**: WebSocket connection is closed. websocketd logs the abnormal termination. websocketd itself does NOT crash. Other connections are unaffected.

---

## PROC-012: Script Not Executable

**Priority**: P1
**Preconditions**: Create a script file WITHOUT execute permission

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./noexec.sh`
2. Connect with wscat

**Expected Result**: Connection fails with an appropriate error. Error is logged. websocketd does not crash.

---

## PROC-013: Script Directory Mode

**Priority**: P0
**Preconditions**: Create directory:
```
scripts/
  echo.sh (executable, echoes stdin)
  count.sh (executable, counts to 5)
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 --dir=scripts/`
2. Connect to `ws://localhost:8080/echo.sh` — send "hello", expect echo
3. Connect to `ws://localhost:8080/count.sh` — expect numbers
4. Connect to `ws://localhost:8080/nonexistent.sh` — expect error

**Expected Result**: Each URL maps to the correct script. Nonexistent scripts return 404.

---

## PROC-014: Script Directory with PATH_INFO

**Priority**: P1
**Preconditions**: Script directory mode

**Steps**:
1. Connect to `ws://localhost:8080/echo.sh/extra/path/info`
2. Check PATH_INFO and SCRIPT_NAME in the process environment

**Expected Result**: SCRIPT_NAME is "/echo.sh". PATH_INFO is "/extra/path/info".

**Notes**: Covered by handler_test.go.

---

## PROC-015: Script with Command Arguments

**Priority**: P1

**Steps**:
1. Start websocketd: `websocketd --port=8080 /bin/echo "hello from args"`
2. Connect with wscat

**Expected Result**: Client receives "hello from args". Arguments are passed to the command correctly.

---

## PROC-016: Process Output Buffering

**Priority**: P1
**Preconditions**: Create a Python script that outputs without newline:
```python
#!/usr/bin/env python3
import sys, time
sys.stdout.write("partial...")
sys.stdout.flush()
time.sleep(1)
sys.stdout.write("complete\n")
sys.stdout.flush()
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./buffer.py`
2. Connect with wscat
3. Observe timing of messages

**Expected Result**: In text mode, the client receives nothing until the newline appears, then gets "partial...complete" as one message. Text mode is line-buffered; newline is the delimiter.

**Notes**: Buffering issues reported in issues #406, #388 (PHP), #400. Many languages buffer stdout when not connected to a terminal. Scripts must flush and use newlines.

---

## PROC-017: Long-Running Process

**Priority**: P1
**Preconditions**: Script that runs indefinitely:
```bash
#!/bin/bash
while true; do echo "alive"; sleep 10; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./heartbeat.sh`
2. Connect with wscat
3. Leave connection open for 30+ minutes
4. Verify messages continue arriving

**Expected Result**: Connection stays open indefinitely. Messages continue arriving. No idle timeouts from websocketd itself.

---

## PROC-018: Rapid Process Exit Before Any Client Input

**Priority**: P1
**Preconditions**: Script that exits immediately:
```bash
#!/bin/bash
echo "goodbye"
exit 0
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./quick.sh`
2. Connect with wscat

**Expected Result**: Client receives "goodbye". Connection closes. No race condition or crash.

---

## PROC-019: Script with Spaces in Path

**Priority**: P2

**Steps**:
1. Create a script at `/tmp/my scripts/echo.sh`
2. Start websocketd: `websocketd --port=8080 "/tmp/my scripts/echo.sh"`
3. Connect and verify

**Expected Result**: Script executes correctly despite spaces in path.

---

## PROC-020: Orphaned Child Processes

**Priority**: P1
**Preconditions**: Create a script that spawns background processes:
```bash
#!/bin/bash
sleep 100 &
sleep 100 &
echo "spawned children"
while IFS= read -r line; do echo "$line"; done
```

**Steps**:
1. Start websocketd: `websocketd --port=8080 ./spawner.sh`
2. Connect, verify "spawned children" received
3. Disconnect
4. After 2 seconds, check for orphaned `sleep` processes: `ps aux | grep sleep`

**Expected Result**: The main script process is terminated. Behavior of grandchild processes depends on OS process group handling. At minimum, websocketd itself should not leak resources.

**Notes**: Process group cleanup varies by OS. On Linux, using process groups can help. This is a known limitation.

---

## PROC-021: --maxforks=0 Means Unlimited

**Priority**: P2

**Steps**:
1. Start websocketd: `websocketd --port=8080 --maxforks=0 cat`
2. Open many connections (50+)

**Expected Result**: All connections are accepted (0 means no limit). System resources are the only constraint.
