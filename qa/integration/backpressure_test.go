package integration

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Tests for bidirectional independence and backpressure after the
// PipeEndpoints split into two independent goroutines.

func TestBACKPRESSURE001_BinaryLargePayload(t *testing.T) {
	// The original binary deadlock test — must still pass with the new architecture.
	// Previously required goroutine-per-Send hack; now works because
	// stdin and stdout are drained by independent goroutines.
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	size := 2 * 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	ws.SendBinary(data)

	total := 0
	for total < size {
		ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, chunk, err := ws.conn.ReadMessage()
		if err != nil {
			t.Fatalf("binary echo failed after %d/%d bytes: %v", total, size, err)
		}
		total += len(chunk)
	}
	if total != size {
		t.Errorf("expected %d bytes total, got %d", size, total)
	}
}

func TestBACKPRESSURE002_ProcessExitMidStream(t *testing.T) {
	// Process exits while client still has the connection open.
	// Both directions should shut down cleanly.
	t.Parallel()
	s := startServer(t, "exit", "0", "done")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("done")
	ws.ExpectClosed()

	// Server should still accept new connections
	ws2 := s.Connect("/")
	defer ws2.Close()
	ws2.ExpectMessage("done")
}

func TestBACKPRESSURE003_WebSocketCloseWhileProcessOutputPending(t *testing.T) {
	// Process generates continuous output. Client disconnects mid-stream.
	// Server should terminate the process cleanly without hanging.
	t.Parallel()
	s := startServer(t, "infinite", "10") // tick every 10ms
	ws := s.Connect("/")

	// Receive a few ticks
	ws.Recv()
	ws.Recv()

	// Abruptly close the WebSocket
	ws.conn.UnderlyingConn().Close()

	// Server should still work after cleanup
	ws2 := s.retryConnect(t, "/", 5*time.Second)
	defer ws2.Close()
	msg := ws2.Recv()
	if msg != "tick" {
		t.Errorf("expected 'tick', got %q", msg)
	}
}

func TestBACKPRESSURE004_BidirectionalTraffic(t *testing.T) {
	// Both directions active simultaneously: process outputs ticks while
	// client sends messages. Verifies the two goroutines are independent.
	t.Parallel()
	s := startServer(t, "multi-line", "ready")
	ws := s.Connect("/")
	defer ws.Close()

	// Receive the initial "ready" message
	ws.ExpectMessage("ready")

	// Send messages and receive echoes concurrently
	var received int32
	done := make(chan struct{})

	// Reader goroutine
	go func() {
		for {
			_, err := ws.RecvTimeout(3 * time.Second)
			if err != nil {
				break
			}
			atomic.AddInt32(&received, 1)
		}
		close(done)
	}()

	// Send 20 messages
	for i := 0; i < 20; i++ {
		ws.Send(fmt.Sprintf("msg-%d", i))
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for echoes to arrive then close
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if int(atomic.LoadInt32(&received)) >= 15 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	ws.Close()
	<-done

	got := int(atomic.LoadInt32(&received))
	if got < 15 {
		t.Errorf("expected at least 15 echoed messages, got %d", got)
	}
}

func TestBACKPRESSURE005_SlowConsumerBackpressure(t *testing.T) {
	// Script reads slowly. Client sends rapidly.
	// With backpressure, this should complete without OOM or goroutine leak.
	// The pipe buffer provides natural throttling.
	t.Parallel()
	s := startServer(t, "slow-start", "100") // 100ms delay then echo
	ws := s.Connect("/")
	defer ws.Close()

	// Wait for "ready"
	ws.ExpectMessage("ready")

	// Send several messages rapidly
	for i := 0; i < 10; i++ {
		ws.Send(fmt.Sprintf("msg-%d", i))
	}

	// Collect responses — they should all come back eventually
	for i := 0; i < 10; i++ {
		msg, err := ws.RecvTimeout(10 * time.Second)
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !strings.HasPrefix(msg, "msg-") {
			t.Errorf("unexpected message: %q", msg)
		}
	}
}

func TestBACKPRESSURE006_TextLargeLineStillWorks(t *testing.T) {
	// Large text line (900KB) should work: child reads the full line
	// before writing output, so stdin and stdout don't overlap.
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	msg := strings.Repeat("X", 900*1024)
	ws.Send(msg)
	got, err := ws.RecvTimeout(10 * time.Second)
	if err != nil {
		t.Fatalf("text echo of 900KB line failed: %v", err)
	}
	if len(got) != len(msg) {
		t.Errorf("expected %d chars, got %d", len(msg), len(got))
	}
}
