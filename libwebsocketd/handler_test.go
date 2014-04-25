// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePathWithScriptDir(t *testing.T) {
	baseDir, _ := ioutil.TempDir("", "websockets")
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
	if err != ScriptNotFoundError {
		t.Error("should fail with script not found")
	}

	// non-existing dir
	_, err = GetURLInfo("/hoohar/bang.sh", config)
	if err == nil {
		t.Error("non-existing dir should fail")
	}
	if err != ScriptNotFoundError {
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
