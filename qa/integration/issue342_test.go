package integration

import (
	"testing"
	"time"
)

// Regression test for GitHub issue #342:
// nil pointer dereference in WebSocketEndpoint.Send when the WebSocket
// connection is broken while the process is still writing output.
// The crash was at websocket_endpoint.go:52 — w.Close() called when w is nil
// because NextWriter returned an error on a dead connection.
func TestIssue342_NilPointerOnBrokenConnection(t *testing.T) {
	t.Parallel()
	// Use a script that outputs continuously — the process will try to send
	// after the client has disconnected.
	s := startServer(t, "infinite", "10") // output every 10ms
	ws := s.Connect("/")

	// Receive a few messages to confirm it's working
	ws.Recv()
	ws.Recv()

	// Abruptly kill the connection (not a clean close)
	ws.conn.UnderlyingConn().Close()

	// Server should still be alive — verify by connecting again.
	// Before the fix, this would panic with nil pointer dereference.
	ws2 := s.retryConnect(t, "/", 5*time.Second)
	defer ws2.Close()
	ws2.Recv() // should get a "tick"
}
