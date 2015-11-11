// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"io"

	"golang.org/x/net/websocket"
)

type WebSocketEndpoint struct {
	ws     *websocket.Conn
	output chan []byte
	log    *LogScope
	bin    bool
}

func NewWebSocketEndpoint(ws *websocket.Conn, bin bool, log *LogScope) *WebSocketEndpoint {
	return &WebSocketEndpoint{
		ws:     ws,
		output: make(chan []byte),
		log:    log,
		bin:    bin,
	}
}

func (we *WebSocketEndpoint) Terminate() {
	we.log.Trace("websocket", "Terminated websocket connection")
}

func (we *WebSocketEndpoint) Output() chan []byte {
	return we.output
}

func (we *WebSocketEndpoint) Send(msg []byte) bool {
	var err error
	if we.bin {
		err = websocket.Message.Send(we.ws, msg)
	} else {
		err = websocket.Message.Send(we.ws, string(msg))
	}
	if err != nil {
		we.log.Trace("websocket", "Cannot send: %s", err)
		return false
	}
	return true
}

func (we *WebSocketEndpoint) StartReading() {
	if we.bin {
		go we.read_binary_frames()
	} else {
		go we.read_text_frames()
	}
}

func (we *WebSocketEndpoint) read_text_frames() {
	for {
		var msg string
		err := websocket.Message.Receive(we.ws, &msg)
		if err != nil {
			if err != io.EOF {
				we.log.Debug("websocket", "Cannot receive: %s", err)
			}
			break
		}
		we.output <- append([]byte(msg), '\n')
	}
	close(we.output)
}

func (we *WebSocketEndpoint) read_binary_frames() {
	for {
		var msg []byte
		err := websocket.Message.Receive(we.ws, &msg)
		if err != nil {
			if err != io.EOF {
				we.log.Debug("websocket", "Cannot receive: %s", err)
			}
			break
		}
		we.output <- msg
	}
	close(we.output)
}
