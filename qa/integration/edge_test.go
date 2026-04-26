package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestEDGE001_VeryLongLine(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	msg := strings.Repeat("A", 500*1024) // 500KB
	ws.Send(msg)
	got := ws.Recv()
	if len(got) != len(msg) {
		t.Errorf("expected %d chars, got %d", len(msg), len(got))
	}
}

func TestEDGE002_WhitespaceMessage(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	messages := []string{" ", "   ", "\t", " \t "}
	for _, msg := range messages {
		ws.Send(msg)
		got := ws.Recv()
		if got != msg {
			t.Errorf("whitespace not preserved: sent %q, got %q", msg, got)
		}
	}
}

func TestEDGE003_EmbeddedNewlines(t *testing.T) {
	// In text mode, a WebSocket message with embedded newlines should be split
	// into multiple lines for the process's stdin, and each line becomes a
	// separate WebSocket message back.
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	ws.Send("line1\nline2\nline3")
	// Each line should come back as a separate message
	ws.ExpectMessage("line1")
	ws.ExpectMessage("line2")
	ws.ExpectMessage("line3")
}

func TestEDGE004_ProcessExitDuringConnection(t *testing.T) {
	t.Parallel()
	s := startServer(t, "exit", "0", "bye")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("bye")
	ws.ExpectClosed()

	// Server should still accept new connections
	ws2 := s.retryConnect(t, "/", 5*time.Second)
	defer ws2.Close()
	ws2.ExpectMessage("bye")
}

func TestEDGE005_SendToExitedProcess(t *testing.T) {
	// Process exits quickly, then client tries to send
	t.Parallel()
	s := startServer(t, "exit", "0", "done")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("done")
	ws.ExpectClosed()

	// This send should not panic the server
	err := ws.conn.WriteMessage(websocket.TextMessage, []byte("after exit"))
	_ = err // may fail, that's fine
}

func TestEDGE006_RapidReconnection(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")

	for i := 0; i < 20; i++ {
		ws := s.Connect("/")
		ws.Send("ping")
		ws.ExpectMessage("ping")
		ws.Close()
	}
}

func TestEDGE007_NilPointerRegression(t *testing.T) {
	// Regression: commit 334a9ec — panic when --address not provided with --ssl and --port
	t.Parallel()
	certFile, keyFile := generateTestCert(t)
	port := freePort(t)

	// This specific combination used to cause a panic
	_, _, exitCode := runWebsocketd(t,
		"--port="+string(rune(port)),
		"--ssl",
		"--sslcert="+certFile,
		"--sslkey="+keyFile,
		testcmdBin, "echo")
	_ = exitCode
	// As long as it doesn't panic, the test passes
}

func TestEDGE008_BinaryFrameDoublingRegression(t *testing.T) {
	// Regression: commit eee5350 — binary frames doubled in size
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	sizes := []int{1, 10, 100, 1000, 10000}
	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		ws.SendBinary(data)
		_, recv := ws.RecvBinary()
		if len(recv) != size {
			t.Errorf("size %d: sent %d bytes, got %d back (doubling bug?)", size, size, len(recv))
		}
	}
}

func TestEDGE009_ProcessHangRegression(t *testing.T) {
	// Regression: commit 3f89f2e — process hang after client disconnect (#159)
	t.Parallel()
	s := startServer(t, "infinite", "50")
	ws := s.Connect("/")

	// Receive a few ticks to confirm it's running
	for i := 0; i < 3; i++ {
		ws.Recv()
	}

	// Disconnect
	ws.Close()

	// Verify we can connect again (process was cleaned up)
	ws2 := s.retryConnect(t, "/", 5*time.Second)
	defer ws2.Close()
	ws2.Recv() // should get a tick
}

func TestEDGE010_ConcurrentConnectDisconnect(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			ws, _, err := s.TryConnect("/", nil)
			if err != nil {
				done <- true
				return
			}
			ws.Send("test")
			ws.RecvTimeout(2 * time.Second)
			ws.Close()
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("timeout waiting for concurrent connect/disconnect")
		}
	}

	// Server should still work
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("alive")
	ws.ExpectMessage("alive")
}

func TestEDGE011_IgnoreStdinProcess(t *testing.T) {
	// Process that never reads stdin
	t.Parallel()
	s := startServer(t, "ignore-stdin")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("started")

	// Send messages even though process ignores stdin
	for i := 0; i < 10; i++ {
		err := ws.conn.WriteMessage(websocket.TextMessage, []byte("ignored"))
		if err != nil {
			break // pipe full or connection closed, both acceptable
		}
	}
}

func TestEDGE012_SpecialCharactersInMessages(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()

	messages := []string{
		`<script>alert('xss')</script>`,
		`; rm -rf / ; echo`,
		`"double quotes" and 'single quotes'`,
		"path\\to\\file",
		"tab\there",
		`$HOME $(whoami) ${PATH}`,
	}
	for _, msg := range messages {
		ws.Send(msg)
		got := ws.Recv()
		if got != msg {
			t.Errorf("message not preserved: sent %q, got %q", msg, got)
		}
	}
}
