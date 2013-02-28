// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"

	"code.google.com/p/go.net/websocket"
)

func main() {
	flag.Usage = PrintHelp
	config := parseCommandLine()

	http.Handle(config.BasePath, websocket.Handler(func(ws *websocket.Conn) {
		acceptWebSocket(ws, &config)
	}))

	if config.Verbose {
		if config.UsingScriptDir {
			log.Print("Listening on ws://", config.Addr, config.BasePath, " -> ", config.ScriptDir)
		} else {
			log.Print("Listening on ws://", config.Addr, config.BasePath, " -> ", config.CommandName, " ", strings.Join(config.CommandArgs, " "))
		}
	}
	err := http.ListenAndServe(config.Addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func acceptWebSocket(ws *websocket.Conn, config *Config) {
	defer ws.Close()

	if config.Verbose {
		log.Print("websocket: CONNECT")
		defer log.Print("websocket: DISCONNECT")
	}

	urlInfo, err := parsePath(ws.Request().URL.Path, config)
	if err != nil {
		// TODO: 404?
		log.Print(err)
		return
	}

	if config.Verbose {
		log.Print("process: URLInfo - ", urlInfo)
	}

	env, err := createEnv(ws, config, urlInfo)
	if err != nil {
		if config.Verbose {
			log.Print("process: Could not setup env: ", err)
		}
		return
	}

	commandName := config.CommandName
	if config.UsingScriptDir {
		commandName = urlInfo.FilePath
	}

	_, stdin, stdout, stderr, err := launchCmd(commandName, config.CommandArgs, env)
	if err != nil {
		if config.Verbose {
			log.Print("process: Failed to start: ", err)
		}
		return
	}

	bufferedStdin := bufio.NewWriter(stdin)

	outbound := make(chan string)
	inbound := make(chan string)

	go readBufferIntoChannel(stdout, outbound, "process stdout", config)
	go readWebsocketIntoChannel(ws, inbound, config)
	go pipeStdErr(stderr, config)
	
LABEL:
	for {
		select {
		case msgFromProcess, ok := <- outbound:
			err := websocket.Message.Send(ws, msgFromProcess)
			if err != nil {
				log.Print("websocket: SENDERROR: ", err)
				stdin.Close()
				break LABEL
			}
			if !ok {
				log.Printf("process terminated")
				break LABEL
			}
		case msgFromSocket, ok  := <- inbound:
			bufferedStdin.WriteString(msgFromSocket)
			bufferedStdin.WriteString("\n")
			bufferedStdin.Flush()
			if !ok {
				log.Printf("websocket closed")
				break LABEL
			}
		}
	}

	
}

func readBufferIntoChannel(input io.ReadCloser, channel chan<- string, name string, config *Config) {
	bufin := bufio.NewReader(input)
	for {
		str, err := bufin.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Unexpected while reading %s: ", name, err)
			} else {
				if config.Verbose {
					log.Printf("%s: CLOSED", name)
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		if config.Verbose {
			log.Printf("%s: OUT : <%s>", name, msg)
		}
		channel <- msg
	}
	close(channel)
}

func readWebsocketIntoChannel(ws *websocket.Conn, channel chan<- string, config *Config) {
	for {
		var msg string
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: RECVERROR: ", err)
				break
			}
			break
		}
		if config.Verbose {
			log.Print("websocket: IN : <", msg, ">")
		}
		channel <- msg
	}
	close(channel)
}

func pipeStdErr(stderr io.ReadCloser, config *Config) {
	bufstderr := bufio.NewReader(stderr)
	for {
		str, err := bufstderr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Unexpected read from process: ", err)
			} else {
				if config.Verbose {
					log.Print("process stderr: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		log.Print("process: STDERR : ", msg)
	}
}
