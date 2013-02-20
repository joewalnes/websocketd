// websocketd: See README
// -Joe Walnes

package websocketd

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"code.google.com/p/go.net/websocket"
)

func main() {
	config := parseCommandLine()

	http.Handle(config.BasePath, websocket.Handler(func(ws *websocket.Conn) {
		acceptWebSocket(ws, &config)
	}))

	if config.Verbose {
		log.Print("Listening on ws://", config.Addr, config.BasePath, " -> ", config.CommandName, " ", strings.Join(config.CommandArgs, " "))
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

	_, stdin, stdout, err := execCmd(config.CommandName, config.CommandArgs)
	if err != nil {
		if config.Verbose {
			log.Print("process: Failed to start: ", err)
		}
		return
	}

	done := make(chan bool)

	outbound := make(chan string, 256)
	go readProcess(stdout, outbound, done, config)
	go writeWebSocket(ws, outbound, done, config)

	inbound := make(chan string, 256)
	go readWebSocket(ws, inbound, done, config)
	go writeProcess(stdin, inbound, done, config)

	<-done
}

func execCmd(commandName string, commandArgs []string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := exec.Command(commandName, commandArgs...)

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
