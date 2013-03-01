// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"code.google.com/p/go.net/websocket"
	"log"
)

type WebsocketEndpoint struct {
	ws     *websocket.Conn
	output chan string
}

func NewWebsocketEndpoint(ws *websocket.Conn) *WebsocketEndpoint {
	return &WebsocketEndpoint{
		ws:     ws,
		output: make(chan string)}
}

func (we *WebsocketEndpoint) Terminate() {
}

func (we *WebsocketEndpoint) Output() chan string {
	return we.output
}

func (we *WebsocketEndpoint) Send(msg string) bool {
	err := websocket.Message.Send(we.ws, msg)
	if err != nil {
		log.Print("websocket: SENDERROR: ", err)
		return false
	}
	return true
}

func (we *WebsocketEndpoint) ReadOutput(config *Config) {
	for {
		var msg string
		err := websocket.Message.Receive(we.ws, &msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: RECVERROR: ", err)
				break
			}
			break
		}
		if config.Verbose {
			log.Print("websocket: IN : <", msg, ">")
		}
		we.output <- msg
	}
	close(we.output)
}
