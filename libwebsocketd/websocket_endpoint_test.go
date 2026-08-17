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
		we := NewWebSocketEndpoint(conn, false, quietLogScope(), 0, 0)
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

// TestWebSocketReadLimit verifies that maxFrameSize bounds inbound messages:
// a frame within the limit is delivered, and one over it closes the
// connection instead of being buffered whole (memory-DoS protection). Without
// SetReadLimit gorilla reads an unbounded message into memory.
func TestWebSocketReadLimit(t *testing.T) {
	const limit = 16

	endpoints := make(chan *WebSocketEndpoint, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		we := NewWebSocketEndpoint(conn, false, quietLogScope(), 0, limit)
		we.StartReading()
		endpoints <- we
	}))
	defer srv.Close()

	client, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()
	we := <-endpoints

	// A message within the limit is delivered (text mode appends a newline).
	if err := client.WriteMessage(websocket.TextMessage, []byte("small")); err != nil {
		t.Fatalf("client send failed: %v", err)
	}
	select {
	case msg, ok := <-we.Output():
		if !ok {
			t.Fatal("output channel closed before delivering the within-limit message")
		}
		if string(msg) != "small\n" {
			t.Fatalf("got %q, want %q", msg, "small\n")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for within-limit message")
	}

	// A message over the limit must not be delivered; readFrames hits
	// ErrReadLimit, closes the output channel, and the connection is torn down.
	oversized := make([]byte, limit+1)
	for i := range oversized {
		oversized[i] = 'x'
	}
	if err := client.WriteMessage(websocket.TextMessage, oversized); err != nil {
		t.Fatalf("client send failed: %v", err)
	}
	select {
	case msg, ok := <-we.Output():
		if ok {
			t.Fatalf("SECURITY: oversized message (%d > %d) was delivered instead of rejected: %d bytes",
				len(oversized), limit, len(msg))
		}
		// channel closed — the oversized frame was refused, as intended.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out; oversized frame neither delivered nor rejected")
	}
}
