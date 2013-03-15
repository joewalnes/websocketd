// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.google.com/p/go.net/websocket"
)

type HttpWsMuxHandler struct {
	Config *Config
	Log    *LogScope
}

// Main HTTP handler. Muxes between WebSocket handler, DevConsole or 404.
func (h HttpWsMuxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hdrs := req.Header
	if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && strings.ToLower(hdrs.Get("Connection")) == "upgrade" {
		// WebSocket
		wsHandler := websocket.Handler(func(ws *websocket.Conn) {
			acceptWebSocket(ws, h.Config, h.Log)
		})
		wsHandler.ServeHTTP(w, req)
	} else if h.Config.DevConsole {
		// Dev console (if enabled)
		content := strings.Replace(ConsoleContent, "{{license}}", License, -1)
		http.ServeContent(w, req, ".html", h.Config.StartupTime, strings.NewReader(content))
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

	log = log.NewLevel(log.LogFunc)
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

	defer process.Terminate()

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
				if !wsEndpoint.Send(msgFromProcess) {
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
				log.Trace("websocket", "WebSocket connection closed")
				return
			}
		}
	}
}
