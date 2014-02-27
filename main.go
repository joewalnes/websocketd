// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joewalnes/websocketd/libwebsocketd"
)

func log(l *libwebsocketd.LogScope, level libwebsocketd.LogLevel, levelName string, category string, msg string, args ...interface{}) {
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
	flag.Usage = PrintHelp
	config := parseCommandLine()

	log := libwebsocketd.RootLogScope(config.LogLevel, log)

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

	http.Handle(config.BasePath, libwebsocketd.HttpWsMuxHandler{
		Config: config.Config,
		Log:    log})

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
	wsschema, httpschema := "ws", "http"
	if config.Ssl {
		wsschema, httpschema = "wss", "https"
	}
	for _, addrSingle := range config.Addr {
		log.Info("server", "Starting WebSocket server   : %s://%s%s", wsschema, addrSingle, config.BasePath)
		if config.DevConsole {
			log.Info("server", "Developer console enabled : %s://%s/", httpschema, addrSingle)
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
	}
	select {
	case err := <-rejects:
		log.Fatal("server", "Can't start server: %s", err)
		os.Exit(3)
	}
}
