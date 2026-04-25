# Performance & Scalability

Tests for concurrent connections, throughput, latency, and resource usage.

---

## Concurrent Connections

### PERF-001: 10 Concurrent Connections

**Priority**: P0

**Steps**:
1. `websocketd --port=8080 cat`
2. Open 10 WebSocket connections simultaneously
3. Send a message on each, collect responses

**Expected Result**: All 10 connections work independently. Each message is echoed back correctly.

---

### PERF-002: 100 Concurrent Connections

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 cat`
2. Open 100 WebSocket connections simultaneously
3. Send a message on each

**Expected Result**: All connections work. Check for any failed connections or dropped messages. Monitor websocketd memory and CPU usage.

---

### PERF-003: 1000 Concurrent Connections

**Priority**: P2

**Steps**:
1. Ensure sufficient file descriptors: `ulimit -n 4096`
2. `websocketd --port=8080 cat`
3. Open 1000 WebSocket connections
4. Send a message on each

**Expected Result**: All connections work (subject to OS limits). Note resource usage. Identify bottleneck point.

**Notes**: Issue #356 — connections maxing out at 150.

---

### PERF-004: Connection Churn (High Turnover)

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 cat`
2. Open and close connections rapidly: 100 connect/disconnect cycles per second
3. Monitor for 5 minutes

**Expected Result**: No resource leaks (goroutines, file descriptors, memory). websocketd remains responsive. No crashes.

---

## Throughput

### PERF-005: Message Throughput (Text Mode)

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 cat`
2. Connect and send 10,000 short messages as fast as possible
3. Measure time for all echoes to return

**Expected Result**: Document throughput in messages/second. No messages lost or reordered.

---

### PERF-006: Message Throughput (Binary Mode)

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --binary cat`
2. Send 10,000 binary messages
3. Measure throughput

**Expected Result**: Document throughput. Binary mode may have different characteristics than text mode.

---

### PERF-007: Large Message Transfer (Text)

**Priority**: P2

**Steps**:
1. Send a 10MB text message (single line)
2. Measure time for echo response
3. Verify data integrity

**Expected Result**: Message transfers successfully. Document time and any memory spike.

---

### PERF-008: Large Message Transfer (Binary)

**Priority**: P2

**Steps**:
1. `websocketd --port=8080 --binary cat`
2. Send a 10MB binary payload
3. Measure time and verify integrity

**Expected Result**: Binary data transfers successfully.

---

## Latency

### PERF-009: Round-Trip Latency

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 cat`
2. Measure time between sending a message and receiving the echo
3. Take 1000 measurements, compute min/avg/max/p99

**Expected Result**: Latency should be low (sub-millisecond for localhost). Document baseline numbers.

---

### PERF-010: First Message Latency

**Priority**: P1

**Steps**:
1. Measure time from WebSocket connection open to first message echo
2. This includes process spawn time

**Expected Result**: Document the overhead of process spawning. Compare with raw WebSocket latency.

**Notes**: Issue #448 — long connection setup time.

---

### PERF-011: Connection Setup Time

**Priority**: P1

**Steps**:
1. Measure time from TCP connect to WebSocket handshake completion
2. Compare with and without --reverselookup

**Expected Result**: Document connection setup time. --reverselookup adds DNS resolution delay.

---

## Resource Usage

### PERF-012: Memory Usage Over Time

**Priority**: P1

**Steps**:
1. Start websocketd, note initial memory usage
2. Open 100 connections, note memory
3. Close all connections, note memory after 30 seconds (allow GC)
4. Repeat 5 times

**Expected Result**: Memory returns to near-baseline after connections close. No steady memory growth (no leaks).

---

### PERF-013: Goroutine Count

**Priority**: P1

**Steps**:
1. If possible, expose Go's pprof or check goroutine count
2. Open 10 connections, check goroutine count
3. Close all, check goroutine count

**Expected Result**: Goroutine count returns to baseline after connections close. No goroutine leaks.

---

### PERF-014: CPU Usage at Idle

**Priority**: P2

**Steps**:
1. Start websocketd with no connections
2. Monitor CPU usage for 5 minutes

**Expected Result**: Near-zero CPU usage when idle.

---

### PERF-015: CPU Usage Under Load

**Priority**: P2

**Steps**:
1. Open 50 connections, each sending 10 messages/second
2. Monitor CPU usage

**Expected Result**: Document CPU usage. Should be reasonable for the workload.

---

## Stress Tests

### PERF-016: Sustained Load

**Priority**: P2

**Steps**:
1. Open 50 connections, each sending 1 message/second
2. Run for 1 hour
3. Monitor for errors, memory growth, connection drops

**Expected Result**: System remains stable. No degradation over time. No resource leaks.

---

### PERF-017: Burst After Idle

**Priority**: P2

**Steps**:
1. Start websocketd and leave idle for 30 minutes
2. Suddenly open 100 connections and send messages

**Expected Result**: System responds correctly after idle period. No startup lag or errors.

---

### PERF-018: --maxforks Under Pressure

**Priority**: P1

**Steps**:
1. `websocketd --port=8080 --maxforks=10 cat`
2. Open 10 connections (at limit)
3. While at limit, attempt 100 more connections rapidly

**Expected Result**: All excess connections get 429. The 10 active connections are unaffected. websocketd handles the rejection load gracefully.

---

### PERF-019: Static File Serving Under WebSocket Load

**Priority**: P2
**Preconditions**: websocketd with --staticdir

**Steps**:
1. Open 50 WebSocket connections
2. Simultaneously serve 1000 static file requests
3. Measure static file response times

**Expected Result**: Static file serving is not significantly impacted by WebSocket connections.

---

### PERF-020: Connection Limit Discovery

**Priority**: P2

**Steps**:
1. Open connections one at a time until failure
2. Note the maximum number achieved
3. Check what resource was exhausted (file descriptors, memory, goroutines)

**Expected Result**: Graceful degradation at the limit. No crash. Document the practical limit for the test environment.
