// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cgi"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var ForkNotAllowedError = errors.New("too many forks active")

// RemoteInfo holds information about remote http client
type RemoteInfo struct {
	Addr, Host, Port string
}

func RemoteDetails(remote string, doLookup bool) (*RemoteInfo, error) {
	addr, port, err := net.SplitHostPort(remote)
	if err != nil {
		return nil, err
	}

	var host string
	if doLookup {
		hosts, err := net.LookupAddr(addr)
		if err != nil || len(hosts) == 0 {
			host = addr
		} else {
			host = hosts[0]
		}
	} else {
		host = addr
	}

	return &RemoteInfo{Addr: addr, Host: host, Port: port}, nil
}

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

	log.Associate("url", h.TellURL("http", req.Host, req.RequestURI))

	remote, err := RemoteDetails(req.RemoteAddr, h.Config.ReverseLookup)
	if err != nil {
		log.Error("session", "Could not understand remote address '%s': %s", req.RemoteAddr, err)
		return
	}
	log.Associate("remote", remote.Host)

	upgradeRe := regexp.MustCompile("(?i)(^|[,\\s])Upgrade($|[,\\s])")

	// WebSocket, limited to size of h.forks
	if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && upgradeRe.MatchString(hdrs.Get("Connection")) {

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

				reqInfo := &requestInfo{id: generateId(), http: req, url: urlInfo, remote: remote}
				log.Associate("id", reqInfo.id)

				wsHandler := websocket.Handler(func(ws *websocket.Conn) {
					acceptWebSocket(reqInfo, ws, h.Config, log)
				})

				wsHandshake := func(wsconf *websocket.Config, req *http.Request) (err error) {
					return checkOrigin(wsconf, req, h.Config, log)
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
		content = strings.Replace(content, "{{addr}}", h.TellURL("ws", req.Host, req.RequestURI), -1)
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

// TellURL is a helper function that changes http to https or ws to wss in case if SSL is used
func (h *HttpWsMuxHandler) TellURL(scheme, host, path string) string {
	if h.Config.Ssl {
		return scheme + "s://" + host + path
	}
	return scheme + "://" + host + path
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

func checkOrigin(wsconf *websocket.Config, req *http.Request, config *Config, log *LogScope) (err error) {
	// check for origin to be correct in future
	// handshaker triggers answering with 403 if error was returned
	// We keep behavior of original handshaker that populates this field
	wsconf.Origin, err = websocket.Origin(wsconf, req)
	if err == nil && wsconf.Origin == nil {
		log.Access("session", "rejected null origin")
		return fmt.Errorf("null origin not allowed")
	}
	if err != nil {
		log.Access("session", "Origin parsing error: %s", err)
		return err
	}
	log.Associate("origin", wsconf.Origin.String())

	// If some origin restrictions are present:
	if config.SameOrigin || config.AllowOrigins != nil {
		originServer, originPort, err := tellHostPort(wsconf.Origin.Host, wsconf.Origin.Scheme == "https")
		if err != nil {
			log.Access("session", "Origin hostname parsing error: %s", err)
			return err
		}
		localServer, localPort, err := tellHostPort(req.Host, req.TLS != nil)
		if err != nil {
			log.Access("session", "Origin hostname parsing error: %s", err)
			return err
		}
		if config.SameOrigin && (originServer != localServer || originPort != localPort) {
			log.Access("session", "Same origin policy mismatch")
			return fmt.Errorf("same origin policy violated")
		}
		if config.AllowOrigins != nil {
			matchFound := false
			for _, allowed := range config.AllowOrigins {
				if pos := strings.Index(allowed, "://"); pos > 0 {
					// allowed schema has to match
					allowedURL, err := url.Parse(allowed)
					if err != nil {
						continue // pass bad URLs in origin list
					}
					if allowedURL.Scheme != wsconf.Origin.Scheme {
						continue // mismatch
					}
					allowed = allowed[pos+3:]
				}
				allowServer, allowPort, err := tellHostPort(allowed, false)
				if err != nil {
					continue // unparseable
				}
				if allowPort == "80" && allowed[len(allowed)-3:] != ":80" {
					// any port is allowed, host names need to match
					matchFound = allowServer == localServer
				} else {
					// exact match of host names and ports
					matchFound = allowServer == localServer && allowPort == localPort
				}
				if matchFound {
					break
				}
			}
			if !matchFound {
				log.Access("session", "Origin is not listed in allowed list")
				return fmt.Errorf("origin list matches were not found")
			}
		}
	}
	return nil
}

func acceptWebSocket(request *requestInfo, ws *websocket.Conn, config *Config, log *LogScope) {
	defer ws.Close()

	log.Access("session", "CONNECT")
	defer log.Access("session", "DISCONNECT")

	env, err := createEnv(request, config, log)
	if err != nil {
		log.Error("process", "Could not create ENV: %s", err)
		return
	}

	commandName := config.CommandName
	if config.UsingScriptDir {
		commandName = request.url.FilePath
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

func tellHostPort(host string, ssl bool) (server, port string, err error) {
	server, port, err = net.SplitHostPort(host)
	if err != nil {
		if addrerr, ok := err.(*net.AddrError); ok && strings.Contains(addrerr.Err, "missing port") {
			server = host
			if ssl {
				port = "443"
			} else {
				port = "80"
			}
			err = nil
		}
	}
	return server, port, err
}
