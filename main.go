// websocketd: See README
// -Joe Walnes

package main

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"path/filepath"
	"os"

	"code.google.com/p/go.net/websocket"
)

func main() {
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

func getScriptPath(ws *websocket.Conn, config *Config) (string, string) {
	if !config.UsingScriptDir {
		return "/", ""
	}
	
	req := ws.Request()
	parts := strings.Split(req.URL.Path, "/")

	path := config.ScriptDir
	var statInfo os.FileInfo
	pathInfo := ""

	for i, p := range parts {
		path = filepath.Join(path, p)
		log.Print("checking if", path, "exists")
		var err error
		statInfo, err = os.Stat(path)
		if err != nil {
			log.Print(path, "does not exist")
			// TODO: 404?
		}
		// end of parts and we are still a dir should fail
		if i == len(parts)-1 && statInfo.IsDir() {
			log.Print("could not find", path)
			// 404?
		}
		if statInfo.IsDir() {
			continue
		} else {
			pathInfo = strings.Join(parts[i:], "/")
			break
		}
	}
	return path, pathInfo
}

func acceptWebSocket(ws *websocket.Conn, config *Config) {
	defer ws.Close()

	if config.Verbose {
		log.Print("websocket: CONNECT")
		defer log.Print("websocket: DISCONNECT")
	}

	path, pathInfo := getScriptPath(ws, config)
	log.Print(path, pathInfo)

	env, err := createEnv(ws, config)
	if err != nil {
		if config.Verbose {
			log.Print("process: Could not setup env: ", err)
		}
		return
	}

	_, stdin, stdout, err := execCmd(config.CommandName, config.CommandArgs, env)
	if err != nil {
		if config.Verbose {
			log.Print("process: Failed to start: ", err)
		}
		return
	}

	done := make(chan bool)

	outbound := make(chan string)
	go readProcess(stdout, outbound, done, config)
	go writeWebSocket(ws, outbound, done, config)

	inbound := make(chan string)
	go readWebSocket(ws, inbound, done, config)
	go writeProcess(stdin, inbound, done, config)

	<-done
}

func execCmd(commandName string, commandArgs []string, env []string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := exec.Command(commandName, commandArgs...)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return cmd, nil, nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return cmd, nil, nil, err
	}

	err = cmd.Start()
	if err != nil {
		return cmd, nil, nil, err
	}

	return cmd, stdin, stdout, err
}
