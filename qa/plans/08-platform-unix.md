# Platform: Unix (macOS, Linux, FreeBSD, Solaris)

Tests for platform-specific behavior across Unix-like operating systems and architectures.

---

## macOS

### MAC-001: macOS Intel (amd64) Basic Functionality

**Priority**: P0
**Preconditions**: macOS with Intel processor, darwin_amd64 build

**Steps**:
1. Build or download websocketd for darwin_amd64
2. `websocketd --port=8080 cat`
3. Connect and verify echo works

**Expected Result**: Full functionality on macOS Intel.

---

### MAC-002: macOS Apple Silicon (arm64)

**Priority**: P0
**Preconditions**: macOS with M1/M2/M3 processor

**Steps**:
1. Build websocketd natively for darwin_arm64: `GOARCH=arm64 go build`
2. `websocketd --port=8080 cat`
3. Connect and verify echo works

**Expected Result**: Full functionality on Apple Silicon.

**Notes**: Issue #425 — segfault on M1 Max. This must be verified.

---

### MAC-003: macOS Bash Scripts

**Priority**: P1

**Steps**:
1. Run the bash examples from `examples/bash/`
2. Verify each works correctly

**Expected Result**: All bash examples function on macOS.

---

### MAC-004: macOS Python Scripts

**Priority**: P1

**Steps**:
1. Run `websocketd --port=8080 python3 examples/python/greeter.py`
2. Connect, send a name, verify greeting response

**Expected Result**: Python 3 scripts work on macOS.

---

## Linux

### LNX-001: Linux amd64 Basic Functionality

**Priority**: P0
**Preconditions**: Linux x86_64, linux_amd64 build

**Steps**:
1. Build or download websocketd for linux_amd64
2. `websocketd --port=8080 cat`
3. Connect and verify

**Expected Result**: Full functionality on Linux amd64.

---

### LNX-002: Linux ARM (Raspberry Pi)

**Priority**: P1
**Preconditions**: Raspberry Pi or ARM device, linux_arm build

**Steps**:
1. Build or download websocketd for linux_arm
2. `websocketd --port=8080 cat`
3. Connect and verify

**Expected Result**: Full functionality on ARM Linux.

**Notes**: Issue #295 — install script doesn't recognize ARM architecture.

---

### LNX-003: Linux ARM64

**Priority**: P1
**Preconditions**: ARM64 Linux system, linux_arm64 build

**Steps**:
1. Build or download websocketd for linux_arm64
2. Basic echo test

**Expected Result**: Full functionality on ARM64 Linux. Commits ac4b25f, 6909932 added ARM64 support.

---

### LNX-004: Linux 386 (32-bit)

**Priority**: P2
**Preconditions**: 32-bit Linux, linux_386 build

**Steps**:
1. Build or download websocketd for linux_386
2. Basic echo test

**Expected Result**: Full functionality on 32-bit Linux.

---

### LNX-005: DEB Package Installation

**Priority**: P2
**Preconditions**: Debian/Ubuntu system

**Steps**:
1. Install the .deb package
2. Verify `websocketd` is in PATH
3. Run `websocketd --version`
4. Basic echo test

**Expected Result**: Package installs cleanly. Binary is accessible. Version matches.

---

### LNX-006: RPM Package Installation

**Priority**: P2
**Preconditions**: RHEL/CentOS/Fedora system

**Steps**:
1. Install the .rpm package
2. Verify `websocketd` is in PATH
3. Basic echo test

**Expected Result**: Package installs cleanly.

---

### LNX-007: Systemd Service

**Priority**: P2
**Preconditions**: Linux with systemd

**Steps**:
1. Create a systemd service file for websocketd
2. Start the service
3. Verify websocketd is running
4. Connect and test
5. Stop the service

**Expected Result**: websocketd works correctly as a systemd service.

**Notes**: Issue #329 — systemd autostart questions.

---

### LNX-008: Signal Handling on Linux

**Priority**: P1

**Steps**:
1. Start websocketd, note its PID
2. Send SIGTERM: `kill -TERM <pid>`
3. Observe behavior

**Expected Result**: websocketd shuts down gracefully. Child processes are terminated.

---

## FreeBSD

### BSD-001: FreeBSD amd64

**Priority**: P2
**Preconditions**: FreeBSD system, freebsd_amd64 build

**Steps**:
1. Build or download websocketd for freebsd_amd64
2. Basic echo test

**Expected Result**: Full functionality on FreeBSD.

---

### BSD-002: FreeBSD 386

**Priority**: P3
**Preconditions**: FreeBSD 32-bit, freebsd_386 build

**Steps**:
1. Build for freebsd_386
2. Basic echo test

**Expected Result**: Full functionality on 32-bit FreeBSD.

---

## OpenBSD

### OBSD-001: OpenBSD amd64

**Priority**: P3
**Preconditions**: OpenBSD system, openbsd_amd64 build

**Steps**:
1. Build for openbsd_amd64
2. Basic echo test

**Expected Result**: Full functionality on OpenBSD.

---

## Solaris

### SOL-001: Solaris amd64

**Priority**: P3
**Preconditions**: Solaris system, solaris_amd64 build

**Steps**:
1. Build for solaris_amd64
2. Basic echo test

**Expected Result**: Full functionality on Solaris.

**Notes**: Issue #419 — Solaris 11 SPARC support request (SPARC not currently built).

---

## Containers

### CTR-001: Docker Container

**Priority**: P1

**Steps**:
1. Create a Dockerfile:
   ```dockerfile
   FROM golang:latest
   COPY . /src
   WORKDIR /src
   RUN go build -o /websocketd
   EXPOSE 8080
   CMD ["/websocketd", "--port=8080", "cat"]
   ```
2. Build and run the container
3. Connect from the host

**Expected Result**: websocketd works inside a Docker container. Port mapping works.

**Notes**: Issue #368 — "the input device is not a TTY" in Docker.

---

### CTR-002: Docker - No TTY

**Priority**: P1

**Steps**:
1. Run websocketd in Docker without `-t` flag (no TTY)
2. Connect and verify functionality

**Expected Result**: websocketd works without a TTY. STDIN/STDOUT piping does not require a terminal.

---

### CTR-003: Termux on Android

**Priority**: P3

**Steps**:
1. Install Go in Termux
2. Build websocketd
3. Basic echo test

**Expected Result**: websocketd builds and runs on Termux.

**Notes**: Issue #452 — Termux + Acode.

---

## Cross-Platform

### XPLAT-001: MIPS Compilation

**Priority**: P3

**Steps**:
1. `GOOS=linux GOARCH=mips go build`
2. Run on MIPS device

**Expected Result**: Compiles and runs on MIPS.

**Notes**: Issue #434 — MIPS compilation support request.

---

### XPLAT-002: Process Group Cleanup

**Priority**: P1

**Steps**:
1. On each platform (macOS, Linux, Windows): run a script that spawns child processes
2. Disconnect the WebSocket client
3. Check for orphaned processes

**Expected Result**: Document the behavior on each platform. Linux/macOS use signals. Windows uses TerminateProcess.

---

### XPLAT-003: File Permissions Across Platforms

**Priority**: P2

**Steps**:
1. On Unix: create a script without execute permission
2. On Windows: create a script (no execute permission concept, but file association matters)
3. Try to use each with websocketd

**Expected Result**: Unix: permission denied error. Windows: may depend on file extension and associations.
