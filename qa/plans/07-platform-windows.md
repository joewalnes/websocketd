# Platform: Windows

Tests specific to Windows behavior, line endings, script types, and process management.

All tests in this file should be run on Windows 10 and Windows 11.

---

## WIN-001: Basic Functionality on Windows

**Priority**: P0
**Preconditions**: websocketd.exe for windows_amd64

**Steps**:
1. Create a batch file `echo.bat`:
   ```batch
   @echo off
   set /p line=
   echo %line%
   ```
2. `websocketd.exe --port=8080 echo.bat`
3. Connect from a WebSocket client

**Expected Result**: WebSocket connection works. Messages are echoed back.

---

## WIN-002: Line Ending Handling (CRLF)

**Priority**: P0
**Preconditions**: Windows with a script that outputs CRLF

**Steps**:
1. Create a batch script that outputs text
2. Connect and observe received WebSocket messages

**Expected Result**: Trailing `\r\n` is stripped by websocketd. Client receives clean text without carriage returns.

**Notes**: Fix in commit 0559afd. The trimEOL function handles both `\n` and `\r\n`.

---

## WIN-003: Batch File (.bat) Scripts

**Priority**: P0

**Steps**:
1. Create various .bat scripts (echo, counter, environment dump)
2. Run each with websocketd
3. Verify correct behavior

**Expected Result**: Batch files execute correctly as WebSocket backends.

---

## WIN-004: PowerShell Scripts

**Priority**: P1

**Steps**:
1. Create a PowerShell script:
   ```powershell
   while ($line = Read-Host) {
       Write-Output "echo: $line"
   }
   ```
2. `websocketd.exe --port=8080 powershell.exe -File echo.ps1`

**Expected Result**: PowerShell scripts work as WebSocket backends.

---

## WIN-005: VBScript Examples

**Priority**: P2

**Steps**:
1. Run the provided VBScript examples from `examples/windows-vbscript/`
2. Connect and verify functionality

**Expected Result**: VBScript examples work correctly on Windows.

---

## WIN-006: JScript Examples

**Priority**: P2

**Steps**:
1. Run the provided JScript examples from `examples/windows-jscript/`
2. Connect and verify functionality

**Expected Result**: JScript examples work correctly on Windows.

---

## WIN-007: Process Termination on Windows

**Priority**: P0

**Steps**:
1. Start websocketd with a long-running script
2. Connect, then disconnect the WebSocket client
3. Check Task Manager for orphaned processes

**Expected Result**: Child process is terminated when WebSocket disconnects. No orphaned processes.

**Notes**: Issue #362 — SIGTERM/SIGINT not supported on Windows. Process termination uses different mechanisms (TerminateProcess).

---

## WIN-008: --maxforks on Windows

**Priority**: P1

**Steps**:
1. `websocketd.exe --port=8080 --maxforks=2 echo.bat`
2. Open 3 connections

**Expected Result**: Third connection gets 429. Fork limiting works on Windows.

---

## WIN-009: Windows Path Handling

**Priority**: P1

**Steps**:
1. Use scripts with Windows-style paths (backslashes)
2. Use script directory mode with Windows paths
3. Test with paths containing spaces

**Expected Result**: Windows paths work correctly. Spaces in paths are handled.

**Notes**: Issue #293 — "is not a valid Win32 application" error.

---

## WIN-010: CGI Scripts on Windows

**Priority**: P1

**Steps**:
1. Create a CGI script (.bat or .exe)
2. `websocketd.exe --port=8080 --cgidir=cgi-bin\ cat`
3. Access the CGI script via HTTP

**Expected Result**: CGI scripts execute on Windows.

**Notes**: Issue #454 — file type requirements for CGI on Windows.

---

## WIN-011: Windows 386 Build

**Priority**: P2

**Steps**:
1. Download the windows_386 build
2. Run on 32-bit Windows
3. Basic echo test

**Expected Result**: 32-bit build works on 32-bit Windows.

**Notes**: Issue #358 — windows amd 64 download issue.

---

## WIN-012: PHP on Windows

**Priority**: P2

**Steps**:
1. `websocketd.exe --port=8080 php.exe script.php`
2. Connect and verify output

**Expected Result**: PHP scripts work on Windows via websocketd.

**Notes**: Issue #320 — PHP background process issues on Windows.

---

## WIN-013: Static Directory on Windows

**Priority**: P1

**Steps**:
1. `websocketd.exe --port=8080 --staticdir=C:\www\static echo.bat`
2. Access static files via HTTP

**Expected Result**: Static files served correctly with Windows paths.

---

## WIN-014: SSL on Windows

**Priority**: P1

**Steps**:
1. Generate certificate on Windows
2. `websocketd.exe --port=8443 --ssl --sslcert=cert.pem --sslkey=key.pem echo.bat`
3. Connect via wss://

**Expected Result**: TLS works on Windows.

---

## WIN-015: Environment Variables on Windows

**Priority**: P1

**Steps**:
1. `set MY_VAR=hello`
2. `websocketd.exe --port=8080 --passenv=MY_VAR set`
3. Connect and check for MY_VAR

**Expected Result**: Environment variables are passed correctly on Windows.
