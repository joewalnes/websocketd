package integration

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Performance tests are skipped in short mode: go test -short ./qa/integration/...
// Run them explicitly: go test -run Perf -count=1 ./qa/integration/...

func TestPERF001_TenConcurrentConnections(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	concurrentEchoTest(t, s, 10)
}

func TestPERF002_FiftyConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()
	s := startServer(t, "echo")
	concurrentEchoTest(t, s, 50)
}

func TestPERF003_MessageThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	count := 1000
	start := time.Now()
	for i := 0; i < count; i++ {
		ws.Send("msg")
	}
	for i := 0; i < count; i++ {
		_, err := ws.RecvTimeout(10 * time.Second)
		if err != nil {
			t.Fatalf("message %d/%d: %v", i+1, count, err)
		}
	}
	elapsed := time.Since(start)
	rate := float64(count) / elapsed.Seconds()
	t.Logf("Throughput: %d messages in %v (%.0f msgs/sec)", count, elapsed, rate)
}

func TestPERF004_ConnectionChurn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()
	s := startServer(t, "echo")

	start := time.Now()
	count := 100
	for i := 0; i < count; i++ {
		ws := s.Connect("/")
		ws.Send("ping")
		ws.ExpectMessage("ping")
		ws.Close()
	}
	elapsed := time.Since(start)
	rate := float64(count) / elapsed.Seconds()
	t.Logf("Connection churn: %d cycles in %v (%.0f conn/sec)", count, elapsed, rate)
}

func TestPERF005_MaxforksUnderPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()
	s := startServerOpts(t, []string{"--maxforks=5"}, "echo")

	// Fill up the fork limit
	var clients []*WSClient
	for i := 0; i < 5; i++ {
		ws := s.Connect("/")
		clients = append(clients, ws)
	}

	// Attempt 50 more connections — all should be rejected
	var rejected int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := s.TryConnect("/", nil)
			if err != nil {
				atomic.AddInt32(&rejected, 1)
			}
		}()
	}
	wg.Wait()

	t.Logf("Rejected %d/50 excess connections under maxforks pressure", rejected)
	if rejected < 40 {
		t.Errorf("expected most connections to be rejected, only %d were", rejected)
	}

	// Existing connections should still work
	for i, ws := range clients {
		ws.Send(fmt.Sprintf("client-%d", i))
		ws.ExpectMessage(fmt.Sprintf("client-%d", i))
		ws.Close()
	}
}

func TestPERF006_LargePayloadBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	// Test increasing binary payload sizes.
	// Note: Very large payloads (1MB+) may time out — binary-echo uses io.Copy
	// which may not flush until EOF or buffer fills.
	sizes := []int{1024, 10 * 1024, 64 * 1024}
	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		start := time.Now()
		ws.SendBinary(data)
		_, recv := ws.RecvBinary()
		elapsed := time.Since(start)

		if len(recv) != size {
			t.Errorf("size %d: expected %d bytes, got %d", size, size, len(recv))
		} else {
			t.Logf("Binary %dKB round-trip: %v", size/1024, elapsed)
		}
	}
}

func TestPERF007_ConcurrentWebSocketAndHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "test.txt", "static content")
	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	var wg sync.WaitGroup

	// 5 WebSocket connections sending messages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ws := s.Connect("/")
			defer ws.Close()
			for j := 0; j < 20; j++ {
				ws.Send(fmt.Sprintf("ws-%d-%d", id, j))
				ws.Recv()
			}
		}(i)
	}

	// Concurrent HTTP requests
	var httpOK int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, body := s.HTTPGetFollow("/test.txt")
			if resp.StatusCode == 200 && strings.Contains(body, "static content") {
				atomic.AddInt32(&httpOK, 1)
			}
		}()
	}

	wg.Wait()
	t.Logf("Concurrent test: %d/50 HTTP requests succeeded alongside 5 WS connections", httpOK)
	if httpOK < 45 {
		t.Errorf("too many HTTP failures: only %d/50 succeeded", httpOK)
	}
}

// Helpers

func concurrentEchoTest(t *testing.T, s *Server, count int) {
	t.Helper()
	var wg sync.WaitGroup
	var failures int32

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ws, _, err := s.TryConnect("/", nil)
			if err != nil {
				atomic.AddInt32(&failures, 1)
				return
			}
			defer ws.Close()
			msg := fmt.Sprintf("conn-%d", id)
			ws.Send(msg)
			got, err := ws.RecvTimeout(10 * time.Second)
			if err != nil || got != msg {
				atomic.AddInt32(&failures, 1)
			}
		}(i)
	}
	wg.Wait()

	if failures > 0 {
		t.Errorf("%d/%d concurrent connections failed", failures, count)
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := writeFile(dir, name, content)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}
