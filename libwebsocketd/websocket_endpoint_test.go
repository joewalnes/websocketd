// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWebSocketTerminateUnblocksParkedReader is the WebSocket-side mirror of
// TestTerminateUnblocksParkedReader: a readFrames goroutine parked on the
// unbuffered output channel send (because the relay stopped draining, e.g.
// the process's stdin write failed) must exit when the endpoint terminates.
// Closing the connection only unblocks NextReader, not a channel send.
func TestWebSocketTerminateUnblocksParkedReader(t *testing.T) {
	before := runtime.NumGoroutine()

	endpoints := make(chan *WebSocketEndpoint, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		we := NewWebSocketEndpoint(conn, false, quietLogScope(), 0)
		we.StartReading()
		endpoints <- we
	}))
	defer srv.Close()

	client, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	we := <-endpoints

	// Never drain we.Output(): the reader picks up this message and parks
	// on the channel send, exactly like a relay whose peer went away.
	if err := client.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("client send failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	we.Terminate()
	client.Close()
	srv.Close()

	// The readFrames goroutine (and the connection's serve goroutines) must
	// exit once the endpoint is terminated and the connection closed.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine leak: %d goroutines before, %d after Terminate (readFrames parked on output send?)",
		before, runtime.NumGoroutine())
}
