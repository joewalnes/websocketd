// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePathWithScriptDir(t *testing.T) {
	baseDir, _ := os.MkdirTemp("", "websockets")
	scriptDir := filepath.Join(baseDir, "foo", "bar")
	scriptPath := filepath.Join(scriptDir, "baz.sh")

	defer os.RemoveAll(baseDir)

	if err := os.MkdirAll(scriptDir, os.ModePerm); err != nil {
		t.Error("could not create ", scriptDir)
	}
	if _, err := os.Create(scriptPath); err != nil {
		t.Error("could not create ", scriptPath)
	}

	config := new(Config)
	config.UsingScriptDir = true
	config.ScriptDir = baseDir

	var res *URLInfo
	var err error

	// simple url
	res, err = GetURLInfo("/foo/bar/baz.sh", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/foo/bar/baz.sh" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "" {
		t.Error("GetURLInfo")
	}
	if res.FilePath != scriptPath {
		t.Error("filePath")
	}

	// url with extra path info
	res, err = GetURLInfo("/foo/bar/baz.sh/some/extra/stuff", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/foo/bar/baz.sh" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "/some/extra/stuff" {
		t.Error("GetURLInfo")
	}
	if res.FilePath != scriptPath {
		t.Error("filePath")
	}

	// non-existing file
	_, err = GetURLInfo("/foo/bar/bang.sh", config)
	if err == nil {
		t.Error("non-existing file should fail")
	}
	if err != ErrScriptNotFound {
		t.Error("should fail with script not found")
	}

	// non-existing dir
	_, err = GetURLInfo("/hoohar/bang.sh", config)
	if err == nil {
		t.Error("non-existing dir should fail")
	}
	if err != ErrScriptNotFound {
		t.Error("should fail with script not found")
	}
}

func TestParsePathExplicitScript(t *testing.T) {
	config := new(Config)
	config.UsingScriptDir = false

	res, err := GetURLInfo("/some/path", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "/some/path" {
		t.Error("GetURLInfo")
	}
	if res.FilePath != "" {
		t.Error("filePath")
	}
}

func TestGetRemoteInfo(t *testing.T) {
	t.Run("TCP address", func(t *testing.T) {
		info, err := GetRemoteInfo("127.0.0.1:54321", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Addr != "127.0.0.1" || info.Port != "54321" {
			t.Errorf("got Addr=%q Port=%q, want 127.0.0.1/54321", info.Addr, info.Port)
		}
	})

	t.Run("Unix domain socket peer", func(t *testing.T) {
		// Go's net/http sets req.RemoteAddr from the accepted conn's
		// RemoteAddr().String(); for a Unix listener the client side is
		// normally unnamed, which Go represents as "@" (or "" in other
		// contexts) - never a "host:port" string. This must not be
		// treated as an error - doing so would refuse every connection
		// over a Unix domain socket.
		for _, remote := range []string{"@", ""} {
			info, err := GetRemoteInfo(remote, false)
			if err != nil {
				t.Fatalf("unexpected error for unix socket peer %q: %v", remote, err)
			}
			if info.Addr != "unix-socket" || info.Host != "unix-socket" {
				t.Errorf("remote %q: got Addr=%q Host=%q, want unix-socket placeholder", remote, info.Addr, info.Host)
			}
			if info.Port != "" {
				t.Errorf("remote %q: expected empty Port for a unix socket peer, got %q", remote, info.Port)
			}
		}
	})
}
