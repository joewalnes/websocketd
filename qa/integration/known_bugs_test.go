package integration

import (
	"testing"
	"time"
)

// These tests assert the EXPECTED (correct) behavior for known bugs.
// They fail against the current code, confirming the bugs exist.
// Each is skipped with an explanation after being verified to fail.

// TestBUG001 removed — fixed: --header now applies to both HTTP and WS responses.

// TestBUG002 removed — fixed: GATEWAY_INTERFACE now set to standard "CGI/1.1" per RFC 3875.

// TestBUG003 removed — not a bug, standard Go http.FileServer path canonicalization.

// TestBUG004 removed — fixed by upgrading gorilla/websocket v1.4.0 → v1.5.3.

// TestBUG005 removed — fixed: go.mod updated from Go 1.15 to Go 1.21.

func TestBUG006_LargeBinaryPayloadStalls(t *testing.T) {
	t.Skip("BUG-006: Binary payloads >64KB deadlock. websocketd writes the full frame to the process's stdin pipe; if it exceeds the OS pipe buffer (~64KB), the write blocks. The process simultaneously tries to write its response to stdout, which also blocks when that pipe buffer fills. Both sides wait on each other — classic pipe deadlock. See qa/bugs-found.md.")

	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	size := 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	ws.SendBinary(data)

	ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, recv, err := ws.conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to receive 1MB binary: %v", err)
	}
	if len(recv) != size {
		t.Errorf("expected %d bytes, got %d", size, len(recv))
	}
}
