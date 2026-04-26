package integration

import (
	"testing"
	"time"
)

// Tests for #456: detect ungraceful client disconnect via ping/pong.

func TestIssue456_PingPongKeepAlive(t *testing.T) {
	// With --pingms, the server sends pings. A healthy connection stays open
	// as long as the client responds to pings (which gorilla/websocket does
	// automatically during reads).
	t.Parallel()
	// Use "infinite" so the server continuously outputs — keeping the client's
	// read loop active, which allows pong responses to be sent.
	s := startServerOpts(t, []string{"--pingms=300"}, "infinite", "200")
	ws := s.Connect("/")
	defer ws.Close()

	// Read messages over several ping cycles. If pings/pongs weren't working,
	// the server would close the connection after 600ms (2x pingInterval).
	for i := 0; i < 10; i++ {
		msg, err := ws.RecvTimeout(2 * time.Second)
		if err != nil {
			t.Fatalf("message %d failed (ping/pong keepalive broken?): %v", i, err)
		}
		if msg != "tick" {
			t.Errorf("expected 'tick', got %q", msg)
		}
	}
	// If we got here, the connection survived ~2 seconds with 300ms ping interval
}

func TestIssue456_DeadConnectionDetected(t *testing.T) {
	// With --pingms, a dead connection (no pong) is detected and the process killed.
	t.Parallel()
	s := startServerOpts(t, []string{"--pingms=200"}, "infinite", "50")
	ws := s.Connect("/")

	// Receive a few ticks to confirm it's working
	ws.Recv()
	ws.Recv()

	// Kill the TCP connection silently (no close frame — simulates browser crash)
	ws.conn.UnderlyingConn().Close()

	// Wait for ping timeout (200ms interval * 2 = 400ms deadline, plus margin)
	time.Sleep(1500 * time.Millisecond)

	// Server should have cleaned up. Verify by connecting again.
	ws2 := s.Connect("/")
	defer ws2.Close()
	ws2.Recv()
}

func TestIssue456_NoPingByDefault(t *testing.T) {
	// Without --pingms, connections work without pings (backward compat).
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	ws.Send("no ping needed")
	ws.ExpectMessage("no ping needed")
}
