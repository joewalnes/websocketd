# Language Examples

Tests for the example scripts across various programming languages to ensure cross-language compatibility.

Each example should be tested by starting websocketd with the example script and verifying correct behavior via a WebSocket client.

---

## Bash

### LANG-001: Bash Greeter

**Priority**: P0
**Preconditions**: Bash available

**Steps**:
1. `websocketd --port=8080 ./examples/bash/greeter.sh`
2. Connect with wscat
3. Send a name

**Expected Result**: Receives a greeting with the name.

---

### LANG-002: Bash Counter

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 ./examples/bash/count.sh`
2. Connect, observe counting

**Expected Result**: Numbers are received as separate WebSocket messages.

---

### LANG-003: Bash Send-Receive

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 ./examples/bash/send-receive.sh`
2. Connect, interact

**Expected Result**: Script reads input and sends responses.

---

### LANG-004: Bash Dump Environment

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 ./examples/bash/dump-env.sh`
2. Connect, observe environment variables

**Expected Result**: CGI environment variables are displayed.

---

### LANG-005: Bash Chat

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 ./examples/bash/chat.sh`
2. Test with multiple connections

**Expected Result**: Chat example functions correctly.

---

## Python

### LANG-010: Python Greeter

**Priority**: P0
**Preconditions**: Python 3 available

**Steps**:
1. `websocketd --port=8080 python3 ./examples/python/greeter.py`
2. Connect, send a name

**Expected Result**: Greeting with the name. Python stdout is unbuffered or flushed.

**Notes**: Python buffers stdout by default when not connected to a terminal. The script must flush or use `python3 -u`.

---

### LANG-011: Python Counter

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 python3 ./examples/python/count.py`
2. Connect, observe

**Expected Result**: Numbers arrive as separate messages.

---

### LANG-012: Python Dump Environment

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 python3 ./examples/python/dump-env.py`
2. Connect, observe

**Expected Result**: Environment variables displayed.

---

### LANG-013: Python Virtual Environment

**Priority**: P2

**Steps**:
1. Create a Python venv
2. Install a package in the venv
3. Run a script that imports the package via websocketd

**Expected Result**: Python venv scripts work with websocketd when PATH is configured.

**Notes**: Issue #408 — ModuleNotFoundError with venv.

---

## Node.js

### LANG-020: Node.js Greeter

**Priority**: P0
**Preconditions**: Node.js installed

**Steps**:
1. `websocketd --port=8080 node ./examples/nodejs/greeter.js`
2. Connect, send a name

**Expected Result**: Greeting response.

---

### LANG-021: Node.js Counter

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 node ./examples/nodejs/count.js`
2. Connect, observe

**Expected Result**: Numbers arrive correctly.

---

### LANG-022: Node.js stdin Handling

**Priority**: P1

**Steps**:
1. Write a Node.js script that reads from process.stdin
2. Run with websocketd
3. Send multiple messages

**Expected Result**: Each WebSocket message arrives as a line on process.stdin.

**Notes**: Issue #442 — Node.js stdin handling issues.

---

## Ruby

### LANG-030: Ruby Greeter

**Priority**: P1
**Preconditions**: Ruby installed

**Steps**:
1. `websocketd --port=8080 ruby ./examples/ruby/greeter.rb`
2. Connect, send a name

**Expected Result**: Greeting response.

---

### LANG-031: Ruby Counter

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 ruby ./examples/ruby/count.rb`
2. Connect, observe

**Expected Result**: Numbers arrive correctly.

---

## Perl

### LANG-040: Perl Greeter

**Priority**: P1
**Preconditions**: Perl installed

**Steps**:
1. `websocketd --port=8080 perl ./examples/perl/greeter.pl`
2. Connect, send a name

**Expected Result**: Greeting response. Perl's autoflush must be enabled.

---

## PHP

### LANG-050: PHP Scripts

**Priority**: P1
**Preconditions**: PHP CLI installed

**Steps**:
1. `websocketd --port=8080 php ./examples/php/greeter.php`
2. Connect, send a name

**Expected Result**: Greeting response.

**Notes**: Issues #406, #388 — "No output from PHP". PHP may require explicit flushing: `ob_implicit_flush(true)` or `fflush(STDOUT)`.

---

## Java

### LANG-060: Java Echo

**Priority**: P2
**Preconditions**: JDK installed

**Steps**:
1. Compile the Java example
2. `websocketd --port=8080 java -cp examples/java Echo`
3. Connect, send a message

**Expected Result**: Message echoed back.

---

## C#/.NET

### LANG-070: C# Example

**Priority**: P2
**Preconditions**: .NET runtime installed

**Steps**:
1. Compile and run the C# example via websocketd

**Expected Result**: Example functions correctly.

---

## Rust

### LANG-080: Rust Example

**Priority**: P2
**Preconditions**: Rust toolchain installed

**Steps**:
1. Compile the Rust example
2. Run via websocketd

**Expected Result**: Compiled Rust binary works as WebSocket backend.

---

## Lua

### LANG-090: Lua Example

**Priority**: P2
**Preconditions**: Lua interpreter installed

**Steps**:
1. `websocketd --port=8080 lua ./examples/lua/greeter.lua`
2. Connect, interact

**Expected Result**: Lua script works correctly. Output is flushed.

---

## QuickJS

### LANG-100: QuickJS Example

**Priority**: P3
**Preconditions**: QuickJS installed

**Steps**:
1. Run the QuickJS examples from `examples/qjs/`

**Expected Result**: QuickJS scripts work correctly.

**Notes**: Added in PR #396.

---

## Swift

### LANG-110: Swift Example

**Priority**: P3
**Preconditions**: Swift installed (macOS or Linux)

**Steps**:
1. Compile and run the Swift example

**Expected Result**: Swift program works as WebSocket backend.

---

## Haskell

### LANG-120: Haskell Example

**Priority**: P3
**Preconditions**: GHC installed

**Steps**:
1. Compile and run the Haskell example

**Expected Result**: Haskell program works correctly.

---

## Cross-Language Concerns

### LANG-200: Stdout Buffering Across Languages

**Priority**: P0

**Steps**:
1. For each language, create a script that outputs a line and does NOT explicitly flush
2. Run with websocketd
3. Check if output is received by the WebSocket client

**Expected Result**: Document which languages require explicit flushing:
- **Needs flush**: Python (use `-u` flag or `flush=True`), PHP (`fflush(STDOUT)`), Perl (`$| = 1`), Java (`System.out.flush()`), C (`fflush(stdout)`)
- **Auto-flush**: Bash (line-buffered by default), Ruby (usually line-buffered)

---

### LANG-201: Exit Code Handling Across Languages

**Priority**: P2

**Steps**:
1. For each language, create a script that exits with code 0 and one that exits with code 1
2. Run each with websocketd
3. Verify connection closure behavior is consistent

**Expected Result**: websocketd handles exit codes consistently regardless of language.

---

### LANG-202: UTF-8 Handling Across Languages

**Priority**: P1

**Steps**:
1. For each language, create a script that echoes back Unicode input
2. Send CJK, emoji, and multi-byte characters
3. Verify correct echo

**Expected Result**: UTF-8 text is preserved through each language's stdin/stdout.

**Notes**: Issue #348 — Chinese character support issues.

---

## CGI Examples

### LANG-300: CGI Script Examples

**Priority**: P1
**Preconditions**: CGI examples from `examples/cgi-bin/`

**Steps**:
1. `websocketd --port=8080 --cgidir=./examples/cgi-bin cat`
2. `curl http://localhost:8080/cgi-bin/<script>`

**Expected Result**: CGI scripts execute correctly and return proper HTTP responses.

---

## HTML Example

### LANG-310: HTML Count Example

**Priority**: P1

**Steps**:
1. Start websocketd with count.sh and --staticdir pointing to examples/html/
2. Open `http://localhost:8080/count.html` in browser

**Expected Result**: HTML page connects via WebSocket and displays counting output.
