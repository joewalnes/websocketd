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
	config, err := parseCommandLine()
	if err == flag.ErrHelp {
		printHelp()
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, err)
		shortHelp()
		os.Exit(1)
	}

	if config.version {
		fmt.Printf("%s %s\n", processName(), getVersionString())
		os.Exit(0)
	}

	if config.license {
		fmt.Printf("%s %s\n", processName(), getVersionString())
		fmt.Printf("%s\n", libwebsocketd.License)
		os.Exit(0)
	}

	service := config.service
	log := libwebsocketd.RootLogScope(service.LogLevel, logfunc)

	if runtime.GOOS != "windows" { // windows relies on env variables to find its libs... e.g. socket stuff
		os.Clearenv() // it's ok to wipe it clean, we already read env variables from passenv into config
	}
	handler := libwebsocketd.NewWebsocketdServer(service, log, config.maxForks)
	http.Handle("/", handler)

	if service.UsingScriptDir {
		log.Info("server", "Serving from directory      : %s", service.ScriptDir)
	} else if service.CommandName != "" {
		log.Info("server", "Serving using application   : %s %s", service.CommandName, strings.Join(service.CommandArgs, " "))
	}
	if service.StaticDir != "" {
		log.Info("server", "Serving static content from : %s", service.StaticDir)
	}
	if service.CgiDir != "" {
		log.Info("server", "Serving CGI scripts from    : %s", service.CgiDir)
	}

	rejects := make(chan error, 1)
	for _, addrSingle := range config.addr {
		log.Info("server", "Starting WebSocket server   : %s", handler.TellURL("ws", addrSingle, "/"))
		if service.DevConsole {
			log.Info("server", "Developer console enabled   : %s", handler.TellURL("http", addrSingle, "/"))
		} else if service.StaticDir != "" || service.CgiDir != "" {
			log.Info("server", "Serving CGI or static files : %s", handler.TellURL("http", addrSingle, "/"))
		}
		// ListenAndServe is blocking function. Let's run it in
		// go routine, reporting result to control channel.
		// Since it's blocking it'll never return non-error.

		go func(addr string) {
			if service.Ssl {
				rejects <- http.ListenAndServeTLS(addr, config.certFile, config.keyFile, nil)
			} else {
				rejects <- http.ListenAndServe(addr, nil)
			}
		}(addrSingle)

		if config.redirPort != 0 {
			go func(addr string) {
				pos := strings.IndexByte(addr, ':')
				rediraddr := addr[:pos] + ":" + strconv.Itoa(config.redirPort) // it would be silly to optimize this one
				redir := &http.Server{Addr: rediraddr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

					// redirect to same hostname as in request but different port and probably schema
					uri := "https://"
					if !service.Ssl {
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
	select {
	case err := <-rejects:
		log.Fatal("server", "Can't start server: %s", err)
		os.Exit(3)
	}
}
