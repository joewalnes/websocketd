package main

import (
	"testing"
	"io/ioutil"
	"path/filepath"
	"os"
)

func TestParsePathWithScriptDir(t *testing.T) {
	baseDir, _	:= ioutil.TempDir("", "websockets")
	scriptDir	:= filepath.Join(baseDir, "foo", "bar")
	scriptPath	:= filepath.Join(scriptDir, "baz.sh")
	
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

	var err error
	var res *URLInfo

	// simple url
	res, err = parsePath("/foo/bar/baz.sh", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/foo/bar/baz.sh" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "" {
		t.Error("pathInfo")
	}
	if res.FilePath != scriptPath {
		t.Error("filePath")
	}

	// url with extra path info
	res, err = parsePath("/foo/bar/baz.sh/some/extra/stuff", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/foo/bar/baz.sh" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "/some/extra/stuff" {
		t.Error("pathInfo")
	}
	if res.FilePath != scriptPath {
		t.Error("filePath")
	}

	// non-existing file
	res, err = parsePath("/foo/bar/bang.sh", config)
	if err == nil {
		t.Error("non-existing file should fail")
	}

	// non-existing dir
	res, err = parsePath("/hoohar/bang.sh", config)
	if err == nil {
		t.Error("non-existing dir should fail")
	}
	
}

func TestParsePathExplicitScript(t *testing.T) {
	config := new(Config)
	config.UsingScriptDir = false

	res, err := parsePath("/some/path", config)
	if err != nil {
		t.Error(err)
	}
	if res.ScriptPath != "/" {
		t.Error("scriptPath")
	}
	if res.PathInfo != "/some/path" {
		t.Error("pathInfo")
	}
	if res.FilePath != "" {
		t.Error("filePath")
	}
}