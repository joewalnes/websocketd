// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketEndpoint struct {
	ws           *websocket.Conn
	output       chan []byte
	log          *LogScope
	mtype        int
	pingInterval time.Duration
}

func NewWebSocketEndpoint(ws *websocket.Conn, bin bool, log *LogScope, pingInterval time.Duration) *WebSocketEndpoint {
	endpoint := &WebSocketEndpoint{
		ws:           ws,
		output:       make(chan []byte),
		log:          log,
		mtype:        websocket.TextMessage,
		pingInterval: pingInterval,
	}
	if bin {
		endpoint.mtype = websocket.BinaryMessage
	}
	return endpoint
}

func (we *WebSocketEndpoint) Terminate() {
	we.ws.Close() // unblocks read_frames goroutine
	we.log.Trace("websocket", "Terminated websocket connection")
}

func (we *WebSocketEndpoint) Output() chan []byte {
	return we.output
}

func (we *WebSocketEndpoint) Send(msg []byte) bool {
	w, err := we.ws.NextWriter(we.mtype)
	if err != nil {
		we.log.Trace("websocket", "Cannot send: %s", err)
		return false
	}

	_, err = w.Write(msg)
	if cerr := w.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		we.log.Trace("websocket", "Cannot send: %s", err)
		return false
	}

	return true
}

func (we *WebSocketEndpoint) StartReading() {
	if we.pingInterval > 0 {
		we.setupPingPong()
	}
	go we.read_frames()
}

// setupPingPong configures ping/pong keepalive to detect dead connections.
// The read deadline is set to 2x the ping interval. Each received pong (or
// any message) resets the deadline. If the client crashes, no pong arrives,
// the deadline expires, and NextReader returns an error.
func (we *WebSocketEndpoint) setupPingPong() {
	readDeadline := we.pingInterval * 2

	// Set initial read deadline
	we.ws.SetReadDeadline(time.Now().Add(readDeadline))

	// When we receive a pong, reset the read deadline.
	// gorilla/websocket calls the PongHandler from the reader goroutine
	// (inside NextReader), so this is safe to call without additional locking.
	we.ws.SetPongHandler(func(string) error {
		we.ws.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// Send pings periodically using WriteControl, which is safe to call
	// concurrently with NextReader (it uses its own write deadline).
	go func() {
		ticker := time.NewTicker(we.pingInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := we.ws.WriteControl(
				websocket.PingMessage, []byte{}, time.Now().Add(we.pingInterval),
			); err != nil {
				return // connection closed
			}
		}
	}()
}

func (we *WebSocketEndpoint) read_frames() {
	for {
		mtype, rd, err := we.ws.NextReader()
		if err != nil {
			we.log.Debug("websocket", "Cannot receive: %s", err)
			break
		}
		if mtype != we.mtype {
			we.log.Debug("websocket", "Received message of type that we did not expect... Ignoring...")
		}

		p, err := io.ReadAll(rd)
		if err != nil && err != io.EOF {
			we.log.Debug("websocket", "Cannot read received message: %s", err)
			break
		}
		switch mtype {
		case websocket.TextMessage:
			we.output <- append(p, '\n')
		case websocket.BinaryMessage:
			we.output <- p
		default:
			we.log.Debug("websocket", "Received message of unknown type: %d", mtype)
		}
	}
	close(we.output)
}
