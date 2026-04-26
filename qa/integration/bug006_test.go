package integration

import (
	"strings"
	"testing"
	"time"
)

// TestBUG006_BinaryDeadlock demonstrates the pipe deadlock in binary mode.
// In binary mode, io.Copy reads a chunk from stdin and immediately writes to stdout.
// With a payload larger than the OS pipe buffer (~64KB), both pipes fill:
// - websocketd blocks writing to stdin (pipe full)
// - child blocks writing to stdout (pipe full)
// - nobody drains either pipe → deadlock
func TestBUG006_BinaryDeadlock(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	size := 2 * 1024 * 1024 // 2MB — well above any OS pipe buffer
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	ws.SendBinary(data)

	ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	total := 0
	for total < size {
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

// TestBUG006_TextLargeLineNoDeadlock confirms that text mode does NOT deadlock
// on a long line, because the child reads the full line before writing output.
// This is NOT the same bug as the binary deadlock.
func TestBUG006_TextLargeLineNoDeadlock(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	// 900KB line — well above pipe buffer, but under testcmd's 1MB scanner limit
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
