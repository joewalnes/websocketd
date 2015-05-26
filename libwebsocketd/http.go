// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"errors"
	"fmt"
	"golang.org/x/net/websocket"
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
)

var ForkNotAllowedError = errors.New("too many forks active")

// WebsocketdServer presents http.Handler interface for requests libwebsocketd is handling.
type WebsocketdServer struct {
	Config *Config
	Log    *LogScope
	forks  chan byte
}

// NewWebsocketdServer creates WebsocketdServer struct with pre-determined config, logscope and maxforks limit
func NewWebsocketdServer(config *Config, log *LogScope, maxforks int) *WebsocketdServer {
	mux := &WebsocketdServer{
		Config: config,
		Log:    log,
	}
	if maxforks > 0 {
		mux.forks = make(chan byte, maxforks)
	}
	return mux
}

// wshandshake returns closure to verify websocket origin header according to configured rules
func (h *WebsocketdServer) wshandshake(log *LogScope) func(*websocket.Config, *http.Request) error {
	return func(wsconf *websocket.Config, req *http.Request) error {
		if len(h.Config.Headers)+len(h.Config.HeadersWs) > 0 {
			if wsconf.Header == nil {
				wsconf.Header = http.Header(make(map[string][]string))
			}
			pushHeaders(wsconf.Header, h.Config.Headers)
			pushHeaders(wsconf.Header, h.Config.HeadersWs)
		}
		return checkOrigin(wsconf, req, h.Config, log)
	}
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

// ServeHTTP muxes between WebSocket handler, CGI handler, DevConsole, Static HTML or 404.
func (h *WebsocketdServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	log := h.Log.NewLevel(h.Log.LogFunc)
	log.Associate("url", h.TellURL("http", req.Host, req.RequestURI))

	pushHeaders(w.Header(), h.Config.Headers)
	if h.Config.CommandName != "" || h.Config.UsingScriptDir {
		hdrs := req.Header
		upgradeRe := regexp.MustCompile("(?i)(^|[,\\s])Upgrade($|[,\\s])")
		// WebSocket, limited to size of h.forks
		if strings.ToLower(hdrs.Get("Upgrade")) == "websocket" && upgradeRe.MatchString(hdrs.Get("Connection")) {
			if h.noteForkCreated() == nil {
				defer h.noteForkCompled()

				handler, err := NewWebsocketdHandler(h, req, log)
				if err != nil {
					if err == ScriptNotFoundError {
						log.Access("session", "NOT FOUND: %s", err)
						http.Error(w, "404 Not Found", 404)
					} else {
						log.Access("session", "INTERNAL ERROR: %s", err)
						http.Error(w, "500 Internal Server Error", 500)
					}
					return
				}

				// Now we are ready for connection upgrade dance...
				wsServer := &websocket.Server{
					Handshake: h.wshandshake(log),
					Handler:   handler.wshandler(log),
				}
				wsServer.ServeHTTP(w, req)
			} else {
				log.Error("http", "Max of possible forks already active, upgrade rejected")
				http.Error(w, "429 Too Many Requests", 429)
			}
			return
		}
	}

	pushHeaders(w.Header(), h.Config.HeadersHTTP)

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

var canonicalHostname string

// TellURL is a helper function that changes http to https or ws to wss in case if SSL is used
func (h *WebsocketdServer) TellURL(scheme, host, path string) string {
	if host[0] == ':' {
		if canonicalHostname == "" {
			var err error
			canonicalHostname, err = os.Hostname()
			if err != nil {
				canonicalHostname = "UNKNOWN"
			}
		}
		host = canonicalHostname + host
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
			return ForkNotAllowedError
		}
	} else {
		return nil
	}
}

func (h *WebsocketdServer) noteForkCompled() {
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
	origin := req.Header.Get("Origin")
	if origin == "" || (origin == "null" && config.AllowOrigins == nil) {
		// we don't want to trust string "null" if there is any
		// enforcements are active
		req.Header.Set("Origin", "file:")
	}

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
					matchFound = allowServer == originServer
				} else {
					// exact match of host names and ports
					matchFound = allowServer == originServer && allowPort == originPort
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
