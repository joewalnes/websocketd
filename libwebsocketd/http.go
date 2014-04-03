// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"code.google.com/p/go.net/websocket"
	"errors"
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

var ForkNotAllowedError = errors.New("too many forks active")

// HttpWsMuxHandler presents http.Handler interface for requests libwebsocketd is handling.
type HttpWsMuxHandler struct {
	Config *Config
	Log    *LogScope
	forks  chan byte
}

// NewHandler creates libwebsocketd mux with given config, logscope and maxforks limit
func NewHandler(config *Config, log *LogScope, maxforks int) *HttpWsMuxHandler {
	mux := &HttpWsMuxHandler{
		Config: config,
		Log:    log,
	}
	if maxforks > 0 {
		mux.forks = make(chan byte, maxforks)
	}
	return mux
}

// ServeHTTP muxes between WebSocket handler, CGI handler, DevConsole, Static HTML or 404.
func (h *HttpWsMuxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log := h.Log.NewLevel(h.Log.LogFunc)

	hdrs := req.Header

	wsschema, httpschema := "ws", "http"
	if h.Config.Ssl {
		wsschema, httpschema = "wss", "https"
	}
	log.Associate("url", fmt.Sprintf("%s://%s%s", httpschema, req.Host, req.URL.RequestURI()))

	_, remoteHost, _, err := remoteDetails(req, h.Config)
	if err != nil {
		log.Error("session", "Could not understand remote address '%s': %s", req.RemoteAddr, err)
		return
	}

	log.Associate("remote", remoteHost)

	upgradeRe := regexp.MustCompile("(?i)(^|[,\\s])Upgrade($|[,\\s])")

	// WebSocket, limited to size of h.forks
	if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && upgradeRe.MatchString(hdrs.Get("Connection")) {
		if hdrs.Get("Origin") == "null" {
			// Fix up mismatch between how Chrome reports Origin
			// when using file:// url (using the string "null"), and
			// how the WebSocket library expects to see it.
			hdrs.Set("Origin", "file:")
		}

		if h.Config.CommandName != "" || h.Config.UsingScriptDir {
			urlInfo, err := parsePath(req.URL.Path, h.Config)
			if err != nil {
				log.Access("session", "NOT FOUND: %s", err)
				http.Error(w, "404 Not Found", 404)
				return
			}
			log.Debug("session", "URLInfo: %s", urlInfo)

			if h.noteForkCreated() == nil {
				defer h.noteForkCompled()
				wsHandler := websocket.Handler(func(ws *websocket.Conn) {
					acceptWebSocket(urlInfo, ws, h.Config, log)
				})

				wsHandshake := func(config *websocket.Config, req *http.Request) error {
					// check for origin to be correct in future
					// handshaker triggers answering with 403 if error was returned we cannot serve 404 out of here
					config.Origin, err = websocket.Origin(config, req)
					if err == nil && config.Origin == nil {
						return fmt.Errorf("null origin")
					}
					return err
				}

				wsServer := websocket.Server{Handler: wsHandler, Handshake: wsHandshake}
				wsServer.ServeHTTP(w, req)
			} else {
				log.Error("http", "Max of possible forks already active, upgrade rejected")
				http.Error(w, "429 Too Many Requests", 429)
			}
			return
		}
	}

	// Dev console (if enabled)
	if h.Config.DevConsole {
		log.Access("http", "DEVCONSOLE")
		content := ConsoleContent
		content = strings.Replace(content, "{{license}}", License, -1)
		content = strings.Replace(content, "{{addr}}", fmt.Sprintf("%s://%s%s", wsschema, req.Host, req.RequestURI), -1)
		http.ServeContent(w, req, ".html", h.Config.StartupTime, strings.NewReader(content))
		return
	}

	// CGI scripts, limited to size of h.forks
	if h.Config.CgiDir != "" {
		filePath := path.Join(h.Config.CgiDir, fmt.Sprintf(".%s", filepath.FromSlash(req.URL.Path)))
		if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {

			log.Associate("cgiscript", filePath)
			if h.noteForkCreated() == nil {
				defer h.noteForkCompled()

				// Make variables to supplement cgi... Environ it uses will show empty list.
				envlen := len(h.Config.ParentEnv)
				cgienv := make([]string, envlen+1)
				if envlen > 0 {
					copy(cgienv, h.Config.ParentEnv)
				}
				cgienv[envlen] = "SERVER_SOFTWARE=" + h.Config.ServerSoftware
				cgiHandler := &cgi.Handler{
					Path: filePath,
					Env: []string{
						"SERVER_SOFTWARE=" + h.Config.ServerSoftware,
					},
				}
				log.Access("http", "CGI")
				cgiHandler.ServeHTTP(w, req)
			} else {
				log.Error("http", "Fork not allowed since maxforks amount has been reached. CGI was not run.")
				http.Error(w, "429 Too Many Requests", 429)
			}
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

func (h *HttpWsMuxHandler) noteForkCreated() error {
	// note that forks can be nil since the construct could've been created by
	// someone who is not using NewHandler
	if h.forks != nil {
		select {
		case h.forks <- 1:
			return nil
		default:
			return ForkNotAllowedError
		}
	} else {
		return nil
	}
}

func (h *HttpWsMuxHandler) noteForkCompled() {
	if h.forks != nil { // see comment in noteForkCreated
		select {
		case <-h.forks:
			return
		default:
			// This could only happen if the completion handler called more times than creation handler above
			// Code should be audited to not allow this to happen, it's desired to have test that would
			// make sure this is impossible but it is not exist yet.
			panic("Cannot deplet number of allowed forks, something is not right in code!")
		}
	}
	return
}

func acceptWebSocket(urlInfo *URLInfo, ws *websocket.Conn, config *Config, log *LogScope) {
	defer ws.Close()

	req := ws.Request()
	id := generateId()

	log.Associate("id", id)
	log.Associate("origin", req.Header.Get("Origin"))

	log.Access("session", "CONNECT")
	defer log.Access("session", "DISCONNECT")

	env, err := createEnv(ws.Request(), config, urlInfo, id, log)
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
