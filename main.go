// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"code.google.com/p/go.net/websocket"
)

func main() {
	flag.Usage = PrintHelp
	config := parseCommandLine()

	http.Handle(config.BasePath, websocket.Handler(func(ws *websocket.Conn) {
		acceptWebSocket(ws, &config)
	}))

	if config.Verbose {
		if config.UsingScriptDir {
			log.Print("Listening on ws://", config.Addr, config.BasePath, " -> ", config.ScriptDir)
		} else {
			log.Print("Listening on ws://", config.Addr, config.BasePath, " -> ", config.CommandName, " ", strings.Join(config.CommandArgs, " "))
		}
	}
	err := http.ListenAndServe(config.Addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func acceptWebSocket(ws *websocket.Conn, config *Config) {
	defer ws.Close()

	if config.Verbose {
		log.Print("websocket: CONNECT")
		defer log.Print("websocket: DISCONNECT")
	}

	urlInfo, err := parsePath(ws.Request().URL.Path, config)
	if err != nil {
		// TODO: 404?
		log.Print(err)
		return
	}

	if config.Verbose {
		log.Print("process: URLInfo - ", urlInfo)
	}

	env, err := createEnv(ws, config, urlInfo)
	if err != nil {
		if config.Verbose {
			log.Print("process: Could not setup env: ", err)
		}
		return
	}

	commandName := config.CommandName
	if config.UsingScriptDir {
		commandName = urlInfo.FilePath
	}

	launched, err := launchCmd(commandName, config.CommandArgs, env)
	if err != nil {
		if config.Verbose {
			log.Print("process: Failed to start: ", err)
		}
		return
	}

	process := NewProcessEndpoint(launched)
	webs := NewWebsocketEndpoint(ws)

	go process.ReadOutput(launched.stdout, config)
	go webs.ReadOutput(config)
	go process.pipeStdErr(config)

	pipeEndpoints(process, webs)
}

func pipeEndpoints(process Endpoint, ws Endpoint) {
	for {
		select {
		case msgFromProcess, ok := <-process.Output():
			sent := ws.Send(msgFromProcess)
			if !sent {
				process.Terminate()
				return
			}
			if !ok {
				log.Printf("process terminated")
				return
			}
		case msgFromSocket, ok := <-ws.Output():
			process.Send(msgFromSocket)
			if !ok {
				process.Terminate()
				log.Printf("websocket closed")
				return
			}
		}
	}
}
