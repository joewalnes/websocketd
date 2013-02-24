// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.net/websocket"
)

const (
	gatewayInterface = "websocketd-CGI/0.1"
)

var headerNewlineToSpace = strings.NewReplacer("\n", " ", "\r", " ")
var headerDashToUnderscore = strings.NewReplacer("-", "_")

func createEnv(ws *websocket.Conn, config *Config) ([]string, error) {
	req := ws.Request()
	headers := req.Header
	url := req.URL

	remoteAddr, remotePort, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil, err
	}

	var remoteHost string
	if config.ReverseLookup {
		remoteHosts, err := net.LookupAddr(remoteAddr)
		if err != nil {
			return nil, err
		}
		if len(remoteHosts) == 0 {
			remoteHost = remoteAddr
		} else {
			remoteHost = remoteHosts[0]
		}
	} else {
		remoteHost = remoteAddr
	}

	serverName, serverPort, err := net.SplitHostPort(req.Host)
	if err != nil {
		return nil, err
	}

	uniqueId := time.Now().UnixNano() // Just use this a unique counter.

	standardEnvCount := 20
	env := make([]string, 0, len(headers)+standardEnvCount)

	// IMPORTANT ---> Adding a header? Make sure standardHeaderCount (above) is up to date.

	// Standard CGI specification headers.
	// As defined in http://tools.ietf.org/html/rfc3875
	env = appendEnv(env, "REMOTE_ADDR", remoteAddr)
	env = appendEnv(env, "REMOTE_HOST", remoteHost)
	env = appendEnv(env, "SERVER_NAME", serverName)
	env = appendEnv(env, "SERVER_PORT", serverPort)
	env = appendEnv(env, "SERVER_PROTOCOL", req.Proto)
	env = appendEnv(env, "SERVER_SOFTWARE", fmt.Sprintf("websocketd/%s", Version()))
	env = appendEnv(env, "GATEWAY_INTERFACE", gatewayInterface)
	env = appendEnv(env, "REQUEST_METHOD", req.Method)
	env = appendEnv(env, "SCRIPT_NAME", "/")
	env = appendEnv(env, "PATH_INFO", url.Path)
	env = appendEnv(env, "PATH_TRANSLATED", url.Path)
	env = appendEnv(env, "QUERY_STRING", url.RawQuery)

	// Not supported, but we explicitly clear them so we don't get leaks from parent environment.
	env = appendEnv(env, "AUTH_TYPE", "")
	env = appendEnv(env, "CONTENT_LENGTH", "")
	env = appendEnv(env, "CONTENT_TYPE", "")
	env = appendEnv(env, "REMOTE_IDENT", "")
	env = appendEnv(env, "REMOTE_USER", "")

	// Non standard, but commonly used headers.
	env = appendEnv(env, "UNIQUE_ID", strconv.FormatInt(uniqueId, 10)) // Based on Apache mod_unique_id.
	env = appendEnv(env, "REMOTE_PORT", remotePort)
	env = appendEnv(env, "REQUEST_URI", url.RequestURI()) // e.g. /foo/blah?a=b

	// The following variables are part of the CGI specification, but are optional
	// and not set by websocketd:
	//
	//   AUTH_TYPE, REMOTE_USER, REMOTE_IDENT
	//     -- Authentication left to the underlying programs.
	//
	//   CONTENT_LENGTH, CONTENT_TYPE
	//     -- makes no sense for WebSocket connections.
	//
	//   HTTPS, SSL_*
	//     -- SSL not supported

	for k, _ := range headers {
		env = appendEnv(env, fmt.Sprintf("HTTP_%s", headerDashToUnderscore.Replace(k)), headers[k]...)
	}

	return env, nil
}

// Adapted from net/http/header.go
func appendEnv(env []string, k string, v ...string) []string {
	if len(v) == 0 {
		return env
	}
	vCleaned := make([]string, 0, len(v))
	for _, val := range v {
		vCleaned = append(vCleaned, strings.TrimSpace(headerNewlineToSpace.Replace(val)))
	}
	return append(env, fmt.Sprintf("%s=%s",
		strings.ToUpper(k),
		strings.Join(vCleaned, ", ")))
}
