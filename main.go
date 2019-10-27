// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/joewalnes/websocketd/libwebsocketd"
)

func logfunc(l *libwebsocketd.LogScope, level libwebsocketd.LogLevel, levelName string, category string, msg string, args ...interface{}) {
	if level < l.MinLevel {
		return
	}
	fullMsg := fmt.Sprintf(msg, args...)

	assocDump := ""
	for index, pair := range l.Associated {
		if index > 0 {
			assocDump += " "
		}
		assocDump += fmt.Sprintf("%s:'%s'", pair.Key, pair.Value)
	}

	l.Mutex.Lock()
	fmt.Printf("%s | %-6s | %-10s | %s | %s\n", libwebsocketd.Timestamp(), levelName, category, assocDump, fullMsg)
	l.Mutex.Unlock()
}

func main() {
	config := parseCommandLine()

	log := libwebsocketd.RootLogScope(config.LogLevel, logfunc)

	if config.DevConsole {
		if config.StaticDir != "" {
			log.Fatal("server", "Invalid parameters: --devconsole cannot be used with --staticdir. Pick one.")
			os.Exit(4)
		}
		if config.CgiDir != "" {
			log.Fatal("server", "Invalid parameters: --devconsole cannot be used with --cgidir. Pick one.")
			os.Exit(4)
		}
	}

	if runtime.GOOS != "windows" { // windows relies on env variables to find its libs... e.g. socket stuff
		os.Clearenv() // it's ok to wipe it clean, we already read env variables from passenv into config
	}
	handler := libwebsocketd.NewWebsocketdServer(config.Config, log, config.MaxForks)
	http.Handle("/", handler)

	if config.UsingScriptDir {
		log.Info("server", "Serving from directory      : %s", config.ScriptDir)
	} else if config.CommandName != "" {
		log.Info("server", "Serving using application   : %s %s", config.CommandName, strings.Join(config.CommandArgs, " "))
	}
	if config.StaticDir != "" {
		log.Info("server", "Serving static content from : %s", config.StaticDir)
	}
	if config.CgiDir != "" {
		log.Info("server", "Serving CGI scripts from    : %s", config.CgiDir)
	}

	rejects := make(chan error, 1)
	for _, addrSingle := range config.Addr {
		log.Info("server", "Starting WebSocket server   : %s", handler.TellURL("ws", addrSingle, "/"))
		if config.DevConsole {
			log.Info("server", "Developer console enabled   : %s", handler.TellURL("http", addrSingle, "/"))
		} else if config.StaticDir != "" || config.CgiDir != "" {
			log.Info("server", "Serving CGI or static files : %s", handler.TellURL("http", addrSingle, "/"))
		}
		// ListenAndServe is blocking function. Let's run it in
		// go routine, reporting result to control channel.
		// Since it's blocking it'll never return non-error.

		go func(addr string) {
			if config.Ssl {
				rejects <- http.ListenAndServeTLS(addr, config.CertFile, config.KeyFile, nil)
			} else {
				rejects <- http.ListenAndServe(addr, nil)
			}
		}(addrSingle)

		if config.RedirPort != 0 {
			go func(addr string) {
				pos := strings.IndexByte(addr, ':')
				rediraddr := addr[:pos] + ":" + strconv.Itoa(config.RedirPort) // it would be silly to optimize this one
				redir := &http.Server{Addr: rediraddr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

					// redirect to same hostname as in request but different port and probably schema
					uri := "https://"
					if !config.Ssl {
						uri = "http://"
					}
					uri += r.Host[:strings.IndexByte(r.Host, ':')] + addr[pos:] + "/"

					http.Redirect(w, r, uri, http.StatusMovedPermanently)
				})}
				log.Info("server", "Starting redirect server   : http://%s/", rediraddr)
				rejects <- redir.ListenAndServe()
			}(addrSingle)
		}
	}
	err := <-rejects
	if err != nil {
		log.Fatal("server", "Can't start server: %s", err)
		os.Exit(3)
	}
}
