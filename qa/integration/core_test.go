package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCORE001_BasicConnection(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestCORE002_MultipleMessages(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()
	for i := 0; i < 10; i++ {
		msg := "message " + strings.Repeat("x", i)
		ws.Send(msg)
		ws.ExpectMessage(msg)
	}
}

func TestCORE003_ServerInitiatedMessages(t *testing.T) {
	t.Parallel()
	s := startServer(t, "count", "5", "10")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessages("1", "2", "3", "4", "5")
}

func TestCORE004_ClientDisconnect(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	ws.Send("hello")
	ws.ExpectMessage("hello")
	ws.Close()
	// Server should handle the disconnect without crashing.
	// Verify by connecting again.
	time.Sleep(100 * time.Millisecond)
	ws2 := s.Connect("/")
	defer ws2.Close()
	ws2.Send("still alive")
	ws2.ExpectMessage("still alive")
}

func TestCORE005_ServerProcessExit(t *testing.T) {
	t.Parallel()
	s := startServer(t, "exit", "0", "goodbye")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessage("goodbye")
	ws.ExpectClosed()
}

func TestCORE006_EmptyMessage(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("")
	ws.ExpectMessage("")
}

func TestCORE007_BinaryModeBasic(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	ws.SendBinary(data)

	msgType, recv := ws.RecvBinary()
	if msgType != websocket.BinaryMessage {
		t.Errorf("expected binary message type, got %d", msgType)
	}
	if len(recv) != len(data) {
		t.Fatalf("binary response length mismatch: sent %d bytes, got %d", len(data), len(recv))
	}
	for i, b := range data {
		if recv[i] != b {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, b, recv[i])
		}
	}
}

func TestCORE008_BinaryFrameNotDoubled(t *testing.T) {
	// Regression test for commit eee5350 — binary frames were doubled in size.
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	ws.SendBinary(data)

	_, recv := ws.RecvBinary()
	if len(recv) != 100 {
		t.Errorf("expected 100 bytes back, got %d (frame doubling regression?)", len(recv))
	}
}

func TestCORE009_TextModeLineBuffering(t *testing.T) {
	t.Parallel()
	s := startServer(t, "output", "line1", "line2", "line3")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessages("line1", "line2", "line3")
}

func TestCORE010_CRLFStripping(t *testing.T) {
	// Regression: commit 0559afd — Windows CRLF handling
	t.Parallel()
	s := startServer(t, "crlf")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessages("line1", "line2", "line3")
}

func TestCORE011_ConcurrentConnections(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			ws := s.Connect("/")
			defer ws.Close()
			msg := "from " + strings.Repeat("x", id)
			ws.Send(msg)
			got := ws.Recv()
			if got != msg {
				t.Errorf("connection %d: expected %q, got %q", id, msg, got)
			}
			done <- true
		}(i)
	}
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for concurrent connections")
		}
	}
}

func TestCORE012_UnicodeMessages(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	messages := []string{
		"Hello 世界",
		"Cześć",
		"日本語テスト",
		"Привет мир",
	}
	for _, msg := range messages {
		ws.Send(msg)
		ws.ExpectMessage(msg)
	}
}

func TestCORE013_LargeTextMessage(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	// 100KB message
	msg := strings.Repeat("A", 100*1024)
	ws.Send(msg)
	got := ws.Recv()
	if len(got) != len(msg) {
		t.Errorf("expected %d chars back, got %d", len(msg), len(got))
	}
}

func TestCORE014_RapidMessages(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	count := 200
	for i := 0; i < count; i++ {
		ws.Send("msg")
	}
	for i := 0; i < count; i++ {
		got, err := ws.RecvTimeout(5 * time.Second)
		if err != nil {
			t.Fatalf("failed to receive message %d/%d: %v", i+1, count, err)
		}
		if got != "msg" {
			t.Errorf("message %d: expected %q, got %q", i, "msg", got)
		}
	}
}

func TestCORE015_WelcomeMessageThenEcho(t *testing.T) {
	t.Parallel()
	s := startServer(t, "welcome", "hello there")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("hello there")
	ws.Send("test")
	ws.ExpectMessage("test")
}

func TestCORE016_WebSocketCloseFrame(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")

	// Send a proper close frame
	ws.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))

	// Should get close frame back or error
	ws.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := ws.conn.ReadMessage()
	if err == nil {
		t.Error("expected error after close, got message")
	}
}

func TestCORE017_ReconnectAfterDisconnect(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")

	for i := 0; i < 10; i++ {
		ws := s.Connect("/")
		ws.Send("hello")
		ws.ExpectMessage("hello")
		ws.Close()
		time.Sleep(20 * time.Millisecond)
	}
}
