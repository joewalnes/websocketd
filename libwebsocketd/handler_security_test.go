package libwebsocketd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckPathBoundary(t *testing.T) {
	t.Run("path within boundary", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "script.sh")
		os.WriteFile(file, []byte("#!/bin/sh"), 0755)

		err := checkPathBoundary(file, dir)
		if err != nil {
			t.Errorf("path within boundary should be allowed: %v", err)
		}
	})

	t.Run("path is boundary itself", func(t *testing.T) {
		dir := t.TempDir()
		err := checkPathBoundary(dir, dir)
		if err != nil {
			t.Errorf("path equal to boundary should be allowed: %v", err)
		}
	})

	t.Run("path outside boundary", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(os.TempDir(), "outside-boundary-test")
		os.WriteFile(outside, []byte("secret"), 0644)
		defer os.Remove(outside)

		err := checkPathBoundary(outside, dir)
		if err == nil {
			t.Error("path outside boundary should be rejected")
		}
	})

	t.Run("symlink escape", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks require admin on Windows")
		}
		dir := t.TempDir()

		// Create a file outside the boundary
		outsideDir := t.TempDir()
		outsideFile := filepath.Join(outsideDir, "secret.sh")
		os.WriteFile(outsideFile, []byte("#!/bin/sh\necho pwned"), 0755)

		// Create a symlink inside the boundary pointing outside
		symlink := filepath.Join(dir, "escape.sh")
		os.Symlink(outsideFile, symlink)

		err := checkPathBoundary(symlink, dir)
		if err == nil {
			t.Error("SECURITY: symlink escaping boundary should be rejected")
		}
	})

	t.Run("symlink within boundary", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks require admin on Windows")
		}
		dir := t.TempDir()

		// Create a real file
		realFile := filepath.Join(dir, "real.sh")
		os.WriteFile(realFile, []byte("#!/bin/sh"), 0755)

		// Create a symlink to the real file (within same dir)
		symlink := filepath.Join(dir, "link.sh")
		os.Symlink(realFile, symlink)

		err := checkPathBoundary(symlink, dir)
		if err != nil {
			t.Errorf("symlink within boundary should be allowed: %v", err)
		}
	})
}
