// websocketd: See README
// -Joe Walnes

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"code.google.com/p/go.net/websocket"
)

type Config struct {
	Addr        string   // TCP address to listen on. e.g. ":1234", "1.2.3.4:1234"
	Verbose     bool     // Verbose logging.
	BasePath    string   // Base URL path. e.g. "/"
	CommandName string   // Command to execute.
	CommandArgs []string // Additional args to pass to command
}

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

func parseCommandLine() Config {
	var config Config

	portFlag := flag.Int("port", 80, "HTTP port to listen on (required)")
	addressFlag := flag.String("address", "0.0.0.0", "Interface to bind to (e.g. 127.0.0.1)")
	basePathFlag := flag.String("basepath", "/", "Base URL path (e.g /)")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	config.Addr = fmt.Sprintf("%s:%d", *addressFlag, *portFlag)
	config.Verbose = *verboseFlag
	config.BasePath = *basePathFlag

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("No executable command specified")
	}
	config.CommandName = args[0]
	config.CommandArgs = flag.Args()[1:]

	return config
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

func readProcess(stdout io.ReadCloser, outbound chan<- string, done chan<- bool, config *Config) {
	bufstdout := bufio.NewReader(stdout)
	for {
		str, err := bufstdout.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Unexpected read from process: ", err)
			} else {
				if config.Verbose {

					log.Print("process: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		if config.Verbose {
			log.Print("process: OUT : <", msg, ">")
		}
		outbound <- msg
	}
	close(outbound)
	done <- true
}

func writeProcess(stdin io.WriteCloser, inbound <-chan string, done chan<- bool, config *Config) {
	bufstdin := bufio.NewWriter(stdin)
	for msg := range inbound {
		bufstdin.WriteString(msg)
		bufstdin.WriteString("\n")
		bufstdin.Flush()
	}
	done <- true
}

func readWebSocket(ws *websocket.Conn, inbound chan<- string, done chan<- bool, config *Config) {
	for {
		var msg string
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: RECVERROR: ", err)
			}
			break
		}
		if config.Verbose {
			log.Print("websocket: IN : <", msg, ">")
		}
		inbound <- msg
	}
	close(inbound)
	done <- true
}

func writeWebSocket(ws *websocket.Conn, outbound <-chan string, done chan<- bool, config *Config) {
	for msg := range outbound {
		err := websocket.Message.Send(ws, msg)
		if err != nil {
			if config.Verbose {
				log.Print("websocket: SENDERROR: ", err)
			}
			break
		}
	}
	done <- true
}
