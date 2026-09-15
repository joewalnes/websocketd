// Copyright 2014 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var ErrScriptNotFound = errors.New("script not found")

// WebsocketdHandler is a single request information and processing structure, it handles WS requests out of all that daemon can handle (static, cgi, devconsole)
type WebsocketdHandler struct {
	server *WebsocketdServer

	Id string
	*RemoteInfo
	*URLInfo
	Env []string

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

func (wsh *WebsocketdHandler) accept(ws *websocket.Conn, log *LogScope) {
	defer ws.Close()

	log.Access("session", "CONNECT")
	defer log.Access("session", "DISCONNECT")

	launched, err := launchCmd(wsh.command, wsh.server.Config.CommandArgs, wsh.Env)
	if err != nil {
		log.Error("process", "Could not launch process %s %s (%s)", wsh.command, strings.Join(wsh.server.Config.CommandArgs, " "), err)
		return
	}

	log.Associate("pid", strconv.Itoa(launched.cmd.Process.Pid))

	raw := wsh.server.Config.Raw
	binary := wsh.server.Config.Binary
	process := NewProcessEndpoint(launched, raw, binary, log, wsh.server.Config.PassStderr)
	if cms := wsh.server.Config.CloseMs; cms != 0 {
		process.closetime += time.Duration(cms) * time.Millisecond
	}
	wsEndpoint := NewWebSocketEndpoint(ws, raw, binary, log, wsh.server.Config.PingInterval, wsh.server.Config.MaxFrameSize)

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
		// Not a "host:port" string — this is expected for Unix domain
		// socket clients, whose peer address has no port (Go reports the
		// unnamed peer as "@", or "" in other contexts). Use a clear,
		// stable placeholder rather than leaking that representation, and
		// don't refuse the connection.
		return &RemoteInfo{Addr: "unix-socket", Host: "unix-socket", Port: ""}, nil
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

	// The net/http ServeMux normally guarantees a leading slash, but
	// WebsocketdServer.ServeHTTP is an exported http.Handler and can be
	// embedded without one - and ""[1:] would panic (audit finding A10).
	if len(path) < 1 || path[0] != '/' {
		return nil, ErrScriptNotFound
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

		// Verify the resolved path stays within the script directory.
		// This prevents symlink attacks where a link inside ScriptDir
		// points to an arbitrary file outside it.
		if err := checkPathBoundary(urlInfo.FilePath, config.ScriptDir); err != nil {
			return nil, ErrScriptNotFound
		}

		// no extra args
		if isLastPart {
			return urlInfo, nil
		}

		// build path info from extra parts of url
		urlInfo.PathInfo = "/" + strings.Join(parts[i+1:], "/")
		return urlInfo, nil
	}
	return nil, fmt.Errorf("could not resolve script for path %q", path)
}

// checkPathBoundary is dirRelation read as a refusal: it reports an error
// unless the file at path really sits inside boundary. handler.go, serveCGI
// and boundedDir.Open all guard a directory with it.
//
// Both arguments are made absolute before their symlinks are resolved, so
// the two are compared in one frame of reference, and the answer depends on
// which directory each argument names rather than on how it is spelled.
// EvalSymlinks preserves the relativeness of its argument, so comparing its
// output for a relative boundary against its output for a path was not
// merely strict, it was meaningless, and it erred in both directions: a "."
// boundary refused everything (nothing EvalSymlinks returns starts with
// "./"), while a ".." boundary accepted anything still spelled with a
// leading "../" — exactly what a symlink pointing out of the tree resolves
// to. That failed OPEN, disclosing a file outside a relative --staticdir
// and executing one outside a relative --cgidir.
//
// filepath.Abs collapses ".." lexically, before any symlink is resolved.
// That is the intended reading for the boundary — the operator's own flag
// value, resolved as --dir always has been — and the path side is that
// boundary joined with an already-rooted, cleaned request path, so it
// carries no ".." for the collapse to misread.
func checkPathBoundary(path, boundary string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return err
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return err
	}
	realBoundary, err := filepath.EvalSymlinks(absBoundary)
	if err != nil {
		return err
	}

	// Fast path, and the one every ordinary request takes: the resolved
	// path is spelled with the resolved boundary as its prefix. No extra
	// syscalls, so the per-request cost is what it has always been. It is
	// an accelerator, not a second policy — anything it accepts, the walk
	// below accepts too, which TestCheckPathBoundaryFastPathImpliesIdentity
	// pins — so the walk only ever runs where this test would have refused.
	if realPath == realBoundary || strings.HasPrefix(realPath, realBoundary+string(filepath.Separator)) {
		return nil
	}

	// Slow path: the rare and attack-shaped case, plus one legitimate case
	// the prefix cannot see. EvalSymlinks resolves symlinks but does not
	// canonicalize the spelling that is left, so two symlink-free absolute
	// paths can still name one directory (see dirRelation). The file is
	// then inside the directory the operator configured, and refusing it
	// 404s their own content, so ask the filesystem which directory each
	// path names instead of asking how it is spelled.
	//
	// Accepting on either route, rather than requiring both to agree, is
	// deliberate: each is sound alone — a string prefix over two resolved
	// paths is proof of containment, and so is finding the boundary
	// directory itself among the resolved path's ancestors — and requiring
	// both would fix nothing, because the bug *is* the prefix failing. The
	// narrow risk in admitting identity is a filesystem that synthesizes or
	// recycles inode numbers (some FUSE and network filesystems); that is
	// the same evidence the exec exclusion has always relied on, it is
	// bounded to an ancestor of an already-resolved path, and nothing a
	// request can spell reaches it.
	bi, err := os.Stat(realBoundary)
	if err != nil {
		return err
	}
	if _, rel := dirRelationResolved(realPath, realBoundary, bi); rel == relInside {
		return nil
	}

	// Anything else is refused, relUnknown included: this is the call site
	// that refuses on the negative, so "cannot tell" has to mean refuse.
	// Two of the other three read it the other way, which is why
	// dirRelation reports three states rather than two (see dirRel).
	return fmt.Errorf("path %q escapes boundary %q", realPath, realBoundary)
}

// generateId produces the per-connection identifier exposed as UNIQUE_ID.
// Crypto-random rather than timestamp-derived: a UnixNano id is guessable
// (one connection's id narrows the next one to nanoseconds) and coarse
// enough to collide under bursts. Falls back to the timestamp only if the
// system CSPRNG is unavailable, which is not a condition worth refusing
// connections over.
func generateId() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}
