// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cgi"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

var ErrForkNotAllowed = errors.New("too many forks active")

var upgradeRe = regexp.MustCompile(`(?i)(^|[,\s])Upgrade($|[,\s])`)

// WebsocketdServer presents http.Handler interface for requests libwebsocketd is handling.
type WebsocketdServer struct {
	Config   *Config
	Log      *LogScope
	forks    chan byte
	hostname string // cached os.Hostname(), computed once at startup
}

// NewWebsocketdServer creates WebsocketdServer struct with pre-determined config, logscope and maxforks limit
func NewWebsocketdServer(config *Config, log *LogScope, maxforks int) *WebsocketdServer {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "UNKNOWN"
	}
	mux := &WebsocketdServer{
		Config:   config,
		Log:      log,
		hostname: hostname,
	}
	if maxforks > 0 {
		mux.forks = make(chan byte, maxforks)
	}
	return mux
}

func splitMimeHeader(s string) (string, string) {
	p := strings.IndexByte(s, ':')
	if p < 0 {
		return s, ""
	}
	key := textproto.CanonicalMIMEHeaderKey(s[:p])

	for p = p + 1; p < len(s); p++ {
		if s[p] != ' ' {
			break
		}
	}
	return key, s[p:]
}

func pushHeaders(h http.Header, hdrs []string) {
	for _, hstr := range hdrs {
		h.Add(splitMimeHeader(hstr))
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request.
func isWebSocketUpgrade(req *http.Request) bool {
	hdrs := req.Header
	return strings.ToLower(hdrs.Get("Upgrade")) == "websocket" &&
		upgradeRe.MatchString(hdrs.Get("Connection"))
}

// ServeHTTP muxes between WebSocket handler, CGI handler, DevConsole, Static HTML or 404.
func (h *WebsocketdServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log := h.Log.NewLevel(h.Log.LogFunc)
	log.Associate("url", h.TellURL("http", req.Host, req.RequestURI))

	if h.serveWebSocket(w, req, log) {
		return
	}

	pushHeaders(w.Header(), h.Config.Headers)
	pushHeaders(w.Header(), h.Config.HeadersHTTP)

	if h.serveDevConsole(w, req, log) {
		return
	}
	if h.serveCGI(w, req, log) {
		return
	}
	if h.serveStatic(w, req, log) {
		return
	}

	log.Access("http", "NOT FOUND")
	http.NotFound(w, req)
}

// serveWebSocket handles WebSocket upgrade requests. Returns true if handled.
func (h *WebsocketdServer) serveWebSocket(w http.ResponseWriter, req *http.Request, log *LogScope) bool {
	if h.Config.CommandName == "" && !h.Config.UsingScriptDir {
		return false
	}
	if !isWebSocketUpgrade(req) {
		return false
	}

	if h.noteForkCreated() != nil {
		log.Error("http", "Max of possible forks already active, upgrade rejected")
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
		return true
	}
	defer h.noteForkCompleted()

	handler, err := NewWebsocketdHandler(h, req, log)
	if err != nil {
		if err == ErrScriptNotFound {
			log.Access("session", "NOT FOUND: %s", err)
			http.Error(w, "404 Not Found", 404)
		} else {
			log.Access("session", "INTERNAL ERROR: %s", err)
			http.Error(w, "500 Internal Server Error", 500)
		}
		return true
	}

	var headers http.Header
	if len(h.Config.Headers)+len(h.Config.HeadersWs) > 0 {
		headers = http.Header(make(map[string][]string))
		pushHeaders(headers, h.Config.Headers)
		pushHeaders(headers, h.Config.HeadersWs)
	}

	upgrader := &websocket.Upgrader{
		HandshakeTimeout: h.Config.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			return checkOrigin(req, h.Config, log) == nil
		},
	}
	conn, err := upgrader.Upgrade(w, req, headers)
	if err != nil {
		log.Access("session", "Unable to Upgrade: %s", err)
		http.Error(w, "500 Internal Error", 500)
		return true
	}

	handler.accept(conn, log)
	return true
}

// serveDevConsole serves the interactive development console. Returns true if handled.
func (h *WebsocketdServer) serveDevConsole(w http.ResponseWriter, req *http.Request, log *LogScope) bool {
	if !h.Config.DevConsole {
		return false
	}
	log.Access("http", "DEVCONSOLE")
	content := strings.Replace(ConsoleContent, "{{addr}}", h.TellURL("ws", req.Host, req.RequestURI), -1)
	http.ServeContent(w, req, ".html", h.Config.StartupTime, strings.NewReader(content))
	return true
}

// serveCGI executes CGI scripts from the configured directory. Returns true if handled.
func (h *WebsocketdServer) serveCGI(w http.ResponseWriter, req *http.Request, log *LogScope) bool {
	if h.Config.CgiDir == "" {
		return false
	}
	filePath := path.Join(h.Config.CgiDir, fmt.Sprintf(".%s", filepath.FromSlash(req.URL.Path)))
	fi, err := os.Stat(filePath)
	if err != nil || fi.IsDir() {
		return false
	}

	log.Associate("cgiscript", filePath)
	if h.noteForkCreated() != nil {
		log.Error("http", "Fork not allowed since maxforks amount has been reached. CGI was not run.")
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
		return true
	}
	defer h.noteForkCompleted()

	// Build extra environment for CGI handler.
	// Go's cgi.Handler sets standard CGI variables (RFC 3875 §4.1)
	// automatically from the HTTP request. Env provides additional
	// variables like SERVER_SOFTWARE and any passed parent env vars.
	envlen := len(h.Config.ParentEnv)
	cgienv := make([]string, envlen+1)
	if envlen > 0 {
		copy(cgienv, h.Config.ParentEnv)
	}
	cgienv[envlen] = "SERVER_SOFTWARE=" + h.Config.ServerSoftware
	cgiHandler := &cgi.Handler{
		Path: filePath,
		Env:  cgienv,
	}
	log.Access("http", "CGI")
	cgiHandler.ServeHTTP(w, req)
	return true
}

// serveStatic serves static files from the configured directory. Returns true if handled.
func (h *WebsocketdServer) serveStatic(w http.ResponseWriter, req *http.Request, log *LogScope) bool {
	if h.Config.StaticDir == "" {
		return false
	}
	log.Access("http", "STATIC")
	http.FileServer(http.Dir(h.Config.StaticDir)).ServeHTTP(w, req)
	return true
}

// TellURL is a helper function that changes http to https or ws to wss in case if SSL is used
func (h *WebsocketdServer) TellURL(scheme, host, path string) string {
	if len(host) > 0 && host[0] == ':' {
		host = h.hostname + host
	}
	if h.Config.Ssl {
		return scheme + "s://" + host + path
	}
	return scheme + "://" + host + path
}

func (h *WebsocketdServer) noteForkCreated() error {
	// note that forks can be nil since the construct could've been created by
	// someone who is not using NewWebsocketdServer
	if h.forks != nil {
		select {
		case h.forks <- 1:
			return nil
		default:
			return ErrForkNotAllowed
		}
	}
	return nil
}

func (h *WebsocketdServer) noteForkCompleted() {
	if h.forks != nil {
		select {
		case <-h.forks:
			return
		default:
			// This should never happen — it means noteForkCompleted was called
			// more times than noteForkCreated. Log rather than crash the server.
			return
		}
	}
}

func checkOrigin(req *http.Request, config *Config, log *LogScope) (err error) {
	origin := req.Header.Get("Origin")
	if origin == "" || (origin == "null" && config.AllowOrigins == nil) {
		origin = "file:"
	}

	originParsed, err := url.ParseRequestURI(origin)
	if err != nil {
		log.Access("session", "Origin parsing error: %s", err)
		return err
	}

	log.Associate("origin", originParsed.String())

	if config.SameOrigin || config.AllowOrigins != nil {
		originServer, originPort, err := tellHostPort(originParsed.Host, originParsed.Scheme == "https")
		if err != nil {
			log.Access("session", "Origin hostname parsing error: %s", err)
			return err
		}
		if config.SameOrigin {
			localServer, localPort, err := tellHostPort(req.Host, req.TLS != nil)
			if err != nil {
				log.Access("session", "Request hostname parsing error: %s", err)
				return err
			}
			if originServer != localServer || originPort != localPort {
				log.Access("session", "Same origin policy mismatch")
				return fmt.Errorf("same origin policy violated")
			}
		}
		if config.AllowOrigins != nil {
			if !matchOrigin(originServer, originPort, originParsed.Scheme, config.AllowOrigins) {
				log.Access("session", "Origin is not listed in allowed list")
				return fmt.Errorf("origin list matches were not found")
			}
		}
	}
	return nil
}

// matchOrigin checks if the given origin server/port/scheme matches any entry
// in the allowed origins list. Extracted for testability.
func matchOrigin(originServer, originPort, originScheme string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if pos := strings.Index(allowed, "://"); pos > 0 {
			allowedURL, err := url.Parse(allowed)
			if err != nil {
				continue
			}
			if allowedURL.Scheme != originScheme {
				continue
			}
			allowed = allowed[pos+3:]
		}
		allowServer, allowPort, err := tellHostPort(allowed, false)
		if err != nil {
			continue
		}
		if allowPort == "80" && (len(allowed) < 3 || allowed[len(allowed)-3:] != ":80") {
			// port defaulted to 80 (not explicitly specified), any port is allowed
			if allowServer == originServer {
				return true
			}
		} else {
			if allowServer == originServer && allowPort == originPort {
				return true
			}
		}
	}
	return false
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
