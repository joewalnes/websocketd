// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"code.google.com/p/go.net/websocket"
)

func readWebSocket(ws *websocket.Conn, inbound chan<- string, done chan<- bool, config *Config) {
	for {
		var msg string
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: RECVERROR: ", err)
			}
			break
		}
		if config.Verbose {
			log.Print("websocket: IN : <", msg, ">")
		}
		inbound <- msg
	}
	close(inbound)
	done <- true
}

func writeWebSocket(ws *websocket.Conn, outbound <-chan string, done chan<- bool, config *Config) {
	for msg := range outbound {
		err := websocket.Message.Send(ws, msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: SENDERROR: ", err)
			}
			break
		}
	}
	done <- true
}
