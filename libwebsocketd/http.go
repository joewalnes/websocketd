// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net/http"
	"net/http/cgi"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type HttpWsMuxHandler struct {
	Config *Config
	Log    *LogScope
}

// Main HTTP handler. Muxes between WebSocket handler, DevConsole or 404.
func (h HttpWsMuxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hdrs := req.Header

	log := h.Log.NewLevel(h.Log.LogFunc)
	log.Associate("url", fmt.Sprintf("http://%s%s", req.RemoteAddr, req.URL.RequestURI()))

	_, remoteHost, _, err := remoteDetails(req, h.Config)
	if err != nil {
		log.Error("session", "Could not understand remote address '%s': %s", req.RemoteAddr, err)
		return
	}

	log.Associate("remote", remoteHost)

	upgradeRe := regexp.MustCompile("(?i)([A-z]\\s*,)?\\s*Upgrade\\s*(,\\s*[A-z])?")

	// WebSocket
	if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && upgradeRe.MatchString(hdrs.Get("Connection")) {

		if hdrs.Get("Origin") == "null" {
			// Fix up mismatch between how Chrome reports Origin
			// when using file:// url (using the string "null"), and
			// how the WebSocket library expects to see it.
			hdrs.Set("Origin", "file:")
		}

		if h.Config.CommandName != "" || h.Config.UsingScriptDir {
			wsHandler := websocket.Handler(func(ws *websocket.Conn) {
				acceptWebSocket(ws, h.Config, log)
			})
			wsHandler.ServeHTTP(w, req)
			return
		}
	}

	// Dev console (if enabled)
	if h.Config.DevConsole {
		content := strings.Replace(ConsoleContent, "{{license}}", License, -1)
		http.ServeContent(w, req, ".html", h.Config.StartupTime, strings.NewReader(content))
		return
	}

	// CGI scripts
	if h.Config.CgiDir != "" {
		filePath := path.Join(h.Config.CgiDir, fmt.Sprintf(".%s", filepath.FromSlash(req.URL.Path)))

		if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
			cgiHandler := &cgi.Handler{
				Path: filePath,
			}
			log.Associate("cgiscript", filePath)
			log.Access("http", "CGI")
			cgiHandler.ServeHTTP(w, req)
			return
		}
	}

	// Static files
	if h.Config.StaticDir != "" {
		handler := http.FileServer(http.Dir(h.Config.StaticDir))
		log.Access("http", "STATIC")
		handler.ServeHTTP(w, req)
		return
	}

	// 404
	log.Access("http", "NOT FOUND")
	http.NotFound(w, req)
}

func acceptWebSocket(ws *websocket.Conn, config *Config, log *LogScope) {
	defer ws.Close()

	req := ws.Request()
	id := generateId()

	log.Associate("id", id)
	log.Associate("origin", req.Header.Get("Origin"))

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
