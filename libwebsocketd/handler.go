package libwebsocketd

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrScriptNotFound = errors.New("Script not found")

// WebsocketdHandler is a single request information and processing structure, it handles WS requests out of all that daemon can handle (static, cgi, devconsole)
type WebsocketdHandler struct {
	server *WebsocketdServer

	Id string
	*RemoteInfo
	*URLInfo // TODO: I cannot find where it's used except in one single place as URLInfo.FilePath
	Env      []string

	command string
}

// NewWebsocketdHandler constructs the handler struct. It also prepares *Info elements and generates process environment.
func NewWebsocketdHandler(s *WebsocketdServer, req *http.Request, log *LogScope) (wsh *WebsocketdHandler, err error) {
	wsh = &WebsocketdHandler{server: s, Id: generateId()}
	log.Associate("id", wsh.Id)

	wsh.RemoteInfo, err = GetRemoteInfo(req.RemoteAddr, s.Config.ReverseLookup)
	if err != nil {
		log.Error("session", "Could not understand remote address '%s': %s", req.RemoteAddr, err)
		return nil, err
	}
	log.Associate("remote", wsh.RemoteInfo.Host)

	wsh.URLInfo, err = GetURLInfo(req.URL.Path, s.Config)
	if err != nil {
		log.Access("session", "NOT FOUND: %s", err)
		return nil, err
	}

	wsh.command = s.Config.CommandName
	if s.Config.UsingScriptDir {
		wsh.command = wsh.URLInfo.FilePath
	}

	log.Associate("command", wsh.command)

	wsh.Env = createEnv(wsh, req, log)

	return wsh, nil
}

// wshandler returns function that executes code with given log context
func (wsh *WebsocketdHandler) wshandler(log *LogScope) websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		wsh.accept(ws, log)
	})
}

// accept connects process and websocket.
func (wsh *WebsocketdHandler) accept(ws *websocket.Conn, log *LogScope) {
	defer ws.Close()

	log.Access("handler", "CONNECT")
	defer log.Access("handler", "DISCONNECT")

	p, output, err := wsh.server.launchServerProcess(wsh.command, wsh.Env, log)
	if err != nil {
		log.Error("handler", "Could not launch process %s %s (%s)", wsh.command, strings.Join(wsh.server.Config.CommandArgs, " "), err)
		return
	}

	/// we need to unsubscribe as soon as we done.
	defer p.Unsubscribe(output)

	// send websocket data to process
	input := make(chan string)
	go func() {
		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				close(input)
				return
			}
			input <- msg
		}
	}()

	for {
		select {
		case msg, ok := <-input:
			if ok {
				err = p.PassInput(msg)
				if err != nil {
					log.Info("handler", "Dropping input message, process input returned %s", err)
				}
			} else {
				log.Info("handler", "Websocket client closed connection....")
				return
			}
		case str, ok := <-output:
			err = websocket.Message.Send(ws, str)
			if err != nil {
				log.Trace("handler", "Process data cannot be passed to websocket due to %s", err)
				return
			}
			if !ok {
				// we might still be able to pass input from websocket to process
				log.Trace("handler", "Process stopped producing results")
				return
			}
		case <-time.After(time.Millisecond * 100):
			// check every 100ms if process has been finished
			if p.cmd.ProcessState != nil {
				log.Trace("handler", "Process ended")
				return
			}
		}
	}
}

// RemoteInfo holds information about remote http client
type RemoteInfo struct {
	Addr, Host, Port string
}

// GetRemoteInfo creates RemoteInfo structure and fills its fields appropriately
func GetRemoteInfo(remote string, doLookup bool) (*RemoteInfo, error) {
	addr, port, err := net.SplitHostPort(remote)
	if err != nil {
		return nil, err
	}

	var host string
	if doLookup {
		hosts, err := net.LookupAddr(addr)
		if err != nil || len(hosts) == 0 {
			host = addr
		} else {
			host = hosts[0]
		}
	} else {
		host = addr
	}

	return &RemoteInfo{Addr: addr, Host: host, Port: port}, nil
}

// URLInfo - structure carrying information about current request and it's mapping to filesystem
type URLInfo struct {
	ScriptPath string
	PathInfo   string
	FilePath   string
}

// GetURLInfo is a function that parses path and provides URL info according to libwebsocketd.Config fields
func GetURLInfo(path string, config *Config) (*URLInfo, error) {
	if !config.UsingScriptDir {
		return &URLInfo{"/", path, ""}, nil
	}

	parts := strings.Split(path[1:], "/")
	urlInfo := &URLInfo{}

	for i, part := range parts {
		urlInfo.ScriptPath = strings.Join([]string{urlInfo.ScriptPath, part}, "/")
		urlInfo.FilePath = filepath.Join(config.ScriptDir, urlInfo.ScriptPath)
		isLastPart := i == len(parts)-1
		statInfo, err := os.Stat(urlInfo.FilePath)

		// not a valid path
		if err != nil {
			return nil, ErrScriptNotFound
		}

		// at the end of url but is a dir
		if isLastPart && statInfo.IsDir() {
			return nil, ErrScriptNotFound
		}

		// we've hit a dir, carry on looking
		if statInfo.IsDir() {
			continue
		}

		// no extra args
		if isLastPart {
			return urlInfo, nil
		}

		// build path info from extra parts of url
		urlInfo.PathInfo = "/" + strings.Join(parts[i+1:], "/")
		return urlInfo, nil
	}
	panic(fmt.Sprintf("GetURLInfo cannot parse path %#v", path))
}

func generateId() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
