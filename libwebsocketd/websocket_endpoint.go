// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"io"
	"io/ioutil"

	"github.com/gorilla/websocket"
)

// CONVERT GORILLA
// This file should be altered to use gorilla's websocket connection type and proper
// message dispatching methods

type WebSocketEndpoint struct {
	ws     *websocket.Conn
	output chan []byte
	log    *LogScope
	mtype  int
}

func NewWebSocketEndpoint(ws *websocket.Conn, bin bool, log *LogScope) *WebSocketEndpoint {
	endpoint := &WebSocketEndpoint{
		ws:     ws,
		output: make(chan []byte),
		log:    log,
		mtype:  websocket.TextMessage,
	}
	if bin {
		endpoint.mtype = websocket.BinaryMessage
	}
	return endpoint
}

func (we *WebSocketEndpoint) Terminate() {
	we.log.Trace("websocket", "Terminated websocket connection")
}

func (we *WebSocketEndpoint) Output() chan []byte {
	return we.output
}

func (we *WebSocketEndpoint) Send(msg []byte) bool {
	w, err := we.ws.NextWriter(we.mtype)
	if err == nil {
		_, err := w.Write(msg)
	}
	w.Close() // could need error handling

	if err != nil {
		we.log.Trace("websocket", "Cannot send: %s", err)
		return false
	}

	return true
}

func (we *WebSocketEndpoint) StartReading() {
	go we.read_frames()
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

		p, err := ioutil.ReadAll(rd)
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
