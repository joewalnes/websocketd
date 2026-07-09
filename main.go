// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
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

// serve listens on the given network ("tcp" or "unix") and address/path and
// runs an HTTP(S) server on it, honoring the Ssl/mutual-TLS config. It blocks
// until the listener errors out.
func serve(network, address string, config *Config, log *libwebsocketd.LogScope) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	if !config.Ssl {
		return http.Serve(listener, nil)
	}
	if config.SslCaFile != "" {
		return serveMutualTLS(listener, config.CertFile, config.KeyFile, config.SslCaFile, log)
	}
	return (&http.Server{}).ServeTLS(listener, config.CertFile, config.KeyFile)
}

// serveMutualTLS runs an HTTPS server on the given listener that requires
// client certificates verified against the given CA file.
func serveMutualTLS(listener net.Listener, certFile, keyFile, caFile string, log *libwebsocketd.LogScope) error {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA file %s: %w", caFile, err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificates from %s", caFile)
	}

	server := &http.Server{
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  caCertPool,
		},
	}
	log.Info("server", "Mutual TLS enabled (client certs verified against %s)", caFile)
	return server.ServeTLS(listener, certFile, keyFile)
}

// serveUnixSocket removes a stale socket file left behind by an unclean
// shutdown (if any) and then serves on it. It only ever removes a path that
// is actually a socket, never an arbitrary file that happens to be there.
func serveUnixSocket(path string, config *Config, log *libwebsocketd.LogScope) error {
	if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove stale socket %s: %w", path, err)
		}
	}
	return serve("unix", path, config, log)
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

	// Buffered for every possible sender (one listener plus one redirect
	// server per address, plus the Unix socket listener if any) so no
	// goroutine blocks on report; main exits on the first error received.
	rejectCap := len(config.Addr) * 2
	if config.UnixSocket != "" {
		rejectCap++
	}
	rejects := make(chan error, rejectCap)
	for _, addrSingle := range config.Addr {
		log.Info("server", "Starting WebSocket server   : %s", handler.TellURL("ws", addrSingle, "/"))
		if config.DevConsole {
			log.Info("server", "Developer console enabled   : %s", handler.TellURL("http", addrSingle, "/"))
		} else if config.StaticDir != "" || config.CgiDir != "" {
			log.Info("server", "Serving CGI or static files : %s", handler.TellURL("http", addrSingle, "/"))
		}
		// serve is a blocking call. Let's run it in a goroutine, reporting
		// the result to the control channel. Since it's blocking it'll
		// never return non-error.

		go func(addr string) {
			rejects <- serve("tcp", addr, config, log)
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
					if cpos := strings.IndexByte(r.Host, ':'); cpos > 0 {
						uri += r.Host[:cpos] + addr[pos:] + "/"
					} else {
						uri += r.Host + addr[pos:] + "/"
					}

					// Not an open redirect: the target is the host the client itself
					// sent, switched to the canonical scheme and port.
					http.Redirect(w, r, uri, http.StatusMovedPermanently) // #nosec G710
				})}
				log.Info("server", "Starting redirect server   : http://%s/", rediraddr)
				rejects <- redir.ListenAndServe()
			}(addrSingle)
		}
	}
	if config.UnixSocket != "" {
		log.Info("server", "Starting WebSocket server   : unix socket at %s", config.UnixSocket)
		go func(path string) {
			rejects <- serveUnixSocket(path, config, log)
		}(config.UnixSocket)
	}
	err := <-rejects
	if err != nil {
		log.Fatal("server", "Can't start server: %s", err)
		os.Exit(3)
	}
}
