package libwebsocketd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var ScriptNotFoundError = errors.New("script not found")

// WebsocketdHandler is a single request information and processing structure, it handles WS requests out of all that daemon can handle (static, cgi, devconsole)
type WebsocketdHandler struct {
	server *WebsocketdServer

	Id string
	*RemoteInfo
	*URLInfo // TODO: I cannot find where it's used except in one single place as URLInfo.FilePath
	Env      []string

	command string
}

// NewWebsocketdHandler constructs the struct and parses all required things in it...
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
		config := wsh.URLInfo.Config
		if config.Muxed {
			wsh.maccept(ws, log)
			return
		}
		wsh.accept(ws, log)
	})
}

func (wsh *WebsocketdHandler) maccept(ws *websocket.Conn, log *LogScope) {
	defer ws.Close()

	log.Access("mux session", "CONNECT")
	defer log.Access("mux session", "DISCONNECT")

	config := wsh.URLInfo.Config
	muxed := MuxedLaunchCmd(wsh, log)

	bin := wsh.server.Config.Binary
	wsEndpoint := NewWebSocketEndpoint(ws, bin, log)

	if len(config.OnConnectPush) > 0 {
		wsEndpoint.Send(config.OnConnectPush)
	}

	MuxedAttach(muxed, ws.Request().RemoteAddr, wsEndpoint)
}

func (wsh *WebsocketdHandler) accept(ws *websocket.Conn, log *LogScope) {
	defer ws.Close()

	log.Access("session", "CONNECT")
	defer log.Access("session", "DISCONNECT")
	config := wsh.URLInfo.Config

	launched, err := launchCmd(wsh.command, wsh.server.Config.CommandArgs, wsh.Env)
	if err != nil {
		log.Error("process", "Could not launch process %s %s (%s)",
			wsh.command, strings.Join(wsh.server.Config.CommandArgs, " "),
			err,
		)
		return
	}

	log.Associate("pid", strconv.Itoa(launched.cmd.Process.Pid))

	bin := wsh.server.Config.Binary
	process := NewProcessEndpoint(launched, bin, log)
	if cms := wsh.server.Config.CloseMs; cms != 0 {
		process.closetime += time.Duration(cms) * time.Millisecond
	}
	wsEndpoint := NewWebSocketEndpoint(ws, bin, log)

	if len(config.OnConnectPush) > 0 {
		wsEndpoint.Send(config.OnConnectPush)
	}

	PipeEndpoints(process, wsEndpoint)
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
	Config     *URLConfig
}

type URLConfig struct {
	OnConnectPush []byte
	Muxed         bool
	BufferSize    int
}

// GetURLInfo is a function that parses path and provides URL info according to libwebsocketd.Config fields
func GetURLInfo(path string, config *Config) (*URLInfo, error) {
	if !config.UsingScriptDir {
		return &URLInfo{"/", path, "", &URLConfig{}}, nil
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
			return nil, ScriptNotFoundError
		}

		// at the end of url but is a dir
		if isLastPart && statInfo.IsDir() {
			return nil, ScriptNotFoundError
		}

		// we've hit a dir, carry on looking
		if statInfo.IsDir() {
			continue
		}

		urlInfo.Config = ReadURLConfig(urlInfo.FilePath)

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

func ReadURLConfig(filepath string) *URLConfig {
	c := &URLConfig{}
	f, err := os.OpenFile(filepath+".config", os.O_RDONLY, 0)
	if err != nil {
		return c
	}
	json.NewDecoder(f).Decode(c)
	f.Close()
	return c
}

func generateId() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
