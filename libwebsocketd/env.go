// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	gatewayInterface = "websocketd-CGI/0.1"
)

var headerNewlineToSpace = strings.NewReplacer("\n", " ", "\r", " ")
var headerDashToUnderscore = strings.NewReplacer("-", "_")

func createEnv(handler *WebsocketdHandler, req *http.Request, log *LogScope) []string {
	headers := req.Header

	url := req.URL

	serverName, serverPort, err := tellHostPort(req.Host, handler.server.Config.Ssl)
	if err != nil {
		// This does mean that we cannot detect port from Host: header... Just keep going with "", guessing is bad.
		log.Debug("env", "Host port detection error: %s", err)
		serverPort = ""
	}

	standardEnvCount := 20
	if handler.server.Config.Ssl {
		standardEnvCount += 1
	}

	parentLen := len(handler.server.Config.ParentEnv)
	env := make([]string, 0, len(headers)+standardEnvCount+parentLen+len(handler.server.Config.Env))

	// This variable could be rewritten from outside
	env = appendEnv(env, "SERVER_SOFTWARE", handler.server.Config.ServerSoftware)

	parentStarts := len(env)
	for _, v := range handler.server.Config.ParentEnv {
		env = append(env, v)
	}

	// IMPORTANT ---> Adding a header? Make sure standardHeaderCount (above) is up to date.

	// Standard CGI specification headers.
	// As defined in http://tools.ietf.org/html/rfc3875
	env = appendEnv(env, "REMOTE_ADDR", handler.RemoteInfo.Addr)
	env = appendEnv(env, "REMOTE_HOST", handler.RemoteInfo.Host)
	env = appendEnv(env, "SERVER_NAME", serverName)
	env = appendEnv(env, "SERVER_PORT", serverPort)
	env = appendEnv(env, "SERVER_PROTOCOL", req.Proto)
	env = appendEnv(env, "GATEWAY_INTERFACE", gatewayInterface)
	env = appendEnv(env, "REQUEST_METHOD", req.Method)
	env = appendEnv(env, "SCRIPT_NAME", handler.URLInfo.ScriptPath)
	env = appendEnv(env, "PATH_INFO", handler.URLInfo.PathInfo)
	env = appendEnv(env, "PATH_TRANSLATED", url.Path)
	env = appendEnv(env, "QUERY_STRING", url.RawQuery)

	// Not supported, but we explicitly clear them so we don't get leaks from parent environment.
	env = appendEnv(env, "AUTH_TYPE", "")
	env = appendEnv(env, "CONTENT_LENGTH", "")
	env = appendEnv(env, "CONTENT_TYPE", "")
	env = appendEnv(env, "REMOTE_IDENT", "")
	env = appendEnv(env, "REMOTE_USER", "")

	// Non standard, but commonly used headers.
	env = appendEnv(env, "UNIQUE_ID", handler.Id) // Based on Apache mod_unique_id.
	env = appendEnv(env, "REMOTE_PORT", handler.RemoteInfo.Port)
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
	//   SSL_*
	//     -- SSL variables are not supported, HTTPS=on added for websocketd running with --ssl

	if handler.server.Config.Ssl {
		env = appendEnv(env, "HTTPS", "on")
	}

	if log.MinLevel == LogDebug {
		for i, v := range env {
			if i >= parentStarts && i < parentLen+parentStarts {
				log.Debug("env", "Parent envvar: %v", v)
			} else {
				log.Debug("env", "Std. variable: %v", v)
			}
		}
	}

	for k, hdrs := range headers {
		header := fmt.Sprintf("HTTP_%s", headerDashToUnderscore.Replace(k))
		env = appendEnv(env, header, hdrs...)
		log.Debug("env", "Header variable %s", env[len(env)-1])
	}

	for _, v := range handler.server.Config.Env {
		env = append(env, v)
		log.Debug("env", "External variable: %s", v)
	}

	return env
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
