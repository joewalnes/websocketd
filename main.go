// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"code.google.com/p/go.net/websocket"
)

func main() {
	flag.Usage = PrintHelp
	config := parseCommandLine()

	log := RootLogScope(logAccess) // TODO: Command line option to change level.

	http.Handle(config.BasePath, HttpWsMuxHandler{
		config: &config,
		log:    log})

	log.Info("server", "Starting WebSocket server : ws://%s%s", config.Addr, config.BasePath)
	if config.DevConsole {
		log.Info("server", "Developer console enabled : http://%s/", config.Addr)
	}
	if config.UsingScriptDir {
		log.Info("server", "Serving from directory    : %s", config.ScriptDir)
	} else {
		log.Info("server", "Serving using application : %s %s", config.CommandName, strings.Join(config.CommandArgs, " "))
	}

	err := http.ListenAndServe(config.Addr, nil)
	if err != nil {
		log.Fatal("server", "Could start server: %s", err)
		os.Exit(3)
	}
}

type HttpWsMuxHandler struct {
	config *Config
	log    *LogScope
}

// Main HTTP handler. Muxes between WebSocket handler, DevConsole or 404.
func (h HttpWsMuxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hdrs := req.Header
	if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && strings.ToLower(hdrs.Get("Connection")) == "upgrade" {
		// WebSocket
		wsHandler := websocket.Handler(func(ws *websocket.Conn) {
			acceptWebSocket(ws, h.config, h.log)
		})
		wsHandler.ServeHTTP(w, req)
	} else if h.config.DevConsole {
		// Dev console (if enabled)
		content := strings.Replace(ConsoleContent, "{{license}}", License, -1)
		http.ServeContent(w, req, ".html", h.config.StartupTime, strings.NewReader(content))
	} else {
		// 404
		http.NotFound(w, req)
	}
}

func acceptWebSocket(ws *websocket.Conn, config *Config, log *LogScope) {
	defer ws.Close()

	req := ws.Request()
	id := generateId()
	_, remoteHost, _, err := remoteDetails(ws, config)
	if err != nil {
		log.Error("session", "Could not understand remote address '%s': %s", req.RemoteAddr, err)
		return
	}

	log = log.NewLevel()
	log.Associate("id", id)
	log.Associate("url", fmt.Sprintf("http://%s%s", req.RemoteAddr, req.URL.RequestURI()))
	log.Associate("origin", req.Header.Get("Origin"))
	log.Associate("remote", remoteHost)

	log.Access("session", "CONNECT")
	defer log.Access("session", "DISCONNECT")

	urlInfo, err := parsePath(ws.Request().URL.Path, config)
	if err != nil {
		log.Access("session", "NOT FOUND: %s", err)
		return
	}
	log.Debug("session", "URLInfo: %s", urlInfo)

	env, err := createEnv(ws, config, urlInfo, id)
	if err != nil {
		log.Error("process", "Could not create ENV: %s", err)
		return
	}

	commandName := config.CommandName
	if config.UsingScriptDir {
		commandName = urlInfo.FilePath
	}
	log.Associate("command", commandName)

	launched, err := launchCmd(commandName, config.CommandArgs, env)
	if err != nil {
		log.Error("process", "Could not launch process %s %s (%s)", commandName, strings.Join(config.CommandArgs, " "), err)
		return
	}

	log.Associate("pid", strconv.Itoa(launched.cmd.Process.Pid))

	process := NewProcessEndpoint(launched, log)
	wsEndpoint := NewWebSocketEndpoint(ws, log)

	go process.ReadOutput(launched.stdout, config)
	go wsEndpoint.ReadOutput(config)
	go process.pipeStdErr(config)

	pipeEndpoints(process, wsEndpoint, log)
}

func pipeEndpoints(process Endpoint, wsEndpoint *WebSocketEndpoint, log *LogScope) {
	for {
		select {
		case msgFromProcess, ok := <-process.Output():
			if ok {
				log.Trace("send<-", "%s", msgFromProcess)
				sent := wsEndpoint.Send(msgFromProcess)
				if !sent {
					process.Terminate()
					return
				}
			} else {
				// TODO: Log exit code. Mechanism differs on different platforms.
				log.Trace("process", "Process terminated")
				return
			}
		case msgFromSocket, ok := <-wsEndpoint.Output():
			if ok {
				log.Trace("recv->", "%s", msgFromSocket)
				process.Send(msgFromSocket)
			} else {
				process.Terminate()
				log.Trace("websocket", "WebSocket connection closed")
				return
			}
		}
	}
}
