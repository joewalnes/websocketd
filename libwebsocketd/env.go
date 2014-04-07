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
	"time"
)

const (
	gatewayInterface = "websocketd-CGI/0.1"
)

type requestInfo struct {
	id     string
	http   *http.Request
	remote *RemoteInfo
	url    *URLInfo
}

var headerNewlineToSpace = strings.NewReplacer("\n", " ", "\r", " ")
var headerDashToUnderscore = strings.NewReplacer("-", "_")

func generateId() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func createEnv(req *requestInfo, config *Config, log *LogScope) ([]string, error) {
	headers := req.http.Header

	url := req.http.URL

	serverName, serverPort, err := tellHostPort(req.http.Host, config.Ssl)
	if err != nil {
		// This does mean that we cannot detect port from Host: header... Just keep going with "", guessing is bad.
		log.Debug("env", "Host port detection error: %s", err)
		serverPort = ""
	}

	standardEnvCount := 20
	if config.Ssl {
		standardEnvCount += 1
	}

	parentLen := len(config.ParentEnv)
	env := make([]string, 0, len(headers)+standardEnvCount+parentLen+len(config.Env))

	// This variable could be rewritten from outside
	env = appendEnv(env, "SERVER_SOFTWARE", config.ServerSoftware)

	parentStarts := len(env)
	for _, v := range config.ParentEnv {
		env = append(env, v)
	}

	// IMPORTANT ---> Adding a header? Make sure standardHeaderCount (above) is up to date.

	// Standard CGI specification headers.
	// As defined in http://tools.ietf.org/html/rfc3875
	env = appendEnv(env, "REMOTE_ADDR", req.remote.Addr)
	env = appendEnv(env, "REMOTE_HOST", req.remote.Host)
	env = appendEnv(env, "SERVER_NAME", serverName)
	env = appendEnv(env, "SERVER_PORT", serverPort)
	env = appendEnv(env, "SERVER_PROTOCOL", req.http.Proto)
	env = appendEnv(env, "GATEWAY_INTERFACE", gatewayInterface)
	env = appendEnv(env, "REQUEST_METHOD", req.http.Method)
	env = appendEnv(env, "SCRIPT_NAME", req.url.ScriptPath)
	env = appendEnv(env, "PATH_INFO", req.url.PathInfo)
	env = appendEnv(env, "PATH_TRANSLATED", url.Path)
	env = appendEnv(env, "QUERY_STRING", url.RawQuery)

	// Not supported, but we explicitly clear them so we don't get leaks from parent environment.
	env = appendEnv(env, "AUTH_TYPE", "")
	env = appendEnv(env, "CONTENT_LENGTH", "")
	env = appendEnv(env, "CONTENT_TYPE", "")
	env = appendEnv(env, "REMOTE_IDENT", "")
	env = appendEnv(env, "REMOTE_USER", "")

	// Non standard, but commonly used headers.
	env = appendEnv(env, "UNIQUE_ID", req.id) // Based on Apache mod_unique_id.
	env = appendEnv(env, "REMOTE_PORT", req.remote.Port)
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

	if config.Ssl {
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

	for _, v := range config.Env {
		env = append(env, v)
		log.Debug("env", "External variable: %s", v)
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
