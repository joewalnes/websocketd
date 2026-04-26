package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePort(t *testing.T) {
	tests := []struct {
		name     string
		portFlag int
		ssl      bool
		want     int
	}{
		{"explicit port", 8080, false, 8080},
		{"explicit port with ssl", 8443, true, 8443},
		{"default http", 0, false, 80},
		{"default https", 0, true, 443},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePort(tt.portFlag, tt.ssl)
			if got != tt.want {
				t.Errorf("resolvePort(%d, %v) = %d, want %d", tt.portFlag, tt.ssl, got, tt.want)
			}
		})
	}
}

func TestResolveAddresses(t *testing.T) {
	tests := []struct {
		name     string
		addrlist []string
		port     int
		want     []string
	}{
		{"no addresses", nil, 8080, []string{":8080"}},
		{"empty list", []string{}, 8080, []string{":8080"}},
		{"single address", []string{"127.0.0.1"}, 8080, []string{"127.0.0.1:8080"}},
		{"multiple addresses", []string{"127.0.0.1", "192.168.1.1"}, 9090, []string{"127.0.0.1:9090", "192.168.1.1:9090"}},
		{"ipv6", []string{"[::1]"}, 8080, []string{"[::1]:8080"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAddresses(tt.addrlist, tt.port)
			if len(got) != len(tt.want) {
				t.Fatalf("resolveAddresses() returned %d addrs, want %d", len(got), len(tt.want))
			}
			for i, addr := range got {
				if addr != tt.want[i] {
					t.Errorf("addr[%d] = %q, want %q", i, addr, tt.want[i])
				}
			}
		})
	}
}

func TestValidateSSL(t *testing.T) {
	tests := []struct {
		name    string
		ssl     bool
		cert    string
		key     string
		wantErr bool
	}{
		{"no ssl, no certs", false, "", "", false},
		{"ssl with both certs", true, "cert.pem", "key.pem", false},
		{"ssl missing cert", true, "", "key.pem", true},
		{"ssl missing key", true, "cert.pem", "", true},
		{"ssl missing both", true, "", "", true},
		{"certs without ssl", false, "cert.pem", "key.pem", true},
		{"cert without ssl", false, "cert.pem", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSSL(tt.ssl, tt.cert, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSSL(%v, %q, %q) error = %v, wantErr %v", tt.ssl, tt.cert, tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestBuildParentEnv(t *testing.T) {
	// Set some env vars for testing
	os.Setenv("TEST_WSD_VAR1", "value1")
	os.Setenv("TEST_WSD_VAR2", "value2")
	defer os.Unsetenv("TEST_WSD_VAR1")
	defer os.Unsetenv("TEST_WSD_VAR2")

	t.Run("passes specified vars", func(t *testing.T) {
		env := buildParentEnv("TEST_WSD_VAR1,TEST_WSD_VAR2")
		found1, found2 := false, false
		for _, e := range env {
			if e == "TEST_WSD_VAR1=value1" {
				found1 = true
			}
			if e == "TEST_WSD_VAR2=value2" {
				found2 = true
			}
		}
		if !found1 || !found2 {
			t.Errorf("expected both vars, got %v", env)
		}
	})

	t.Run("skips HTTPS", func(t *testing.T) {
		os.Setenv("HTTPS", "on")
		defer os.Unsetenv("HTTPS")
		env := buildParentEnv("HTTPS,TEST_WSD_VAR1")
		for _, e := range env {
			if e == "HTTPS=on" {
				t.Error("HTTPS should be filtered out")
			}
		}
	})

	t.Run("skips nonexistent vars", func(t *testing.T) {
		env := buildParentEnv("NONEXISTENT_WSD_VAR")
		if len(env) != 0 {
			t.Errorf("expected empty env, got %v", env)
		}
	})

	t.Run("strips newlines from values", func(t *testing.T) {
		os.Setenv("TEST_WSD_NEWLINE", "value\nwith\nnewlines")
		defer os.Unsetenv("TEST_WSD_NEWLINE")
		env := buildParentEnv("TEST_WSD_NEWLINE")
		if len(env) != 1 {
			t.Fatalf("expected 1 env, got %d", len(env))
		}
		if env[0] != "TEST_WSD_NEWLINE=value with newlines" {
			t.Errorf("newlines not cleaned: %q", env[0])
		}
	})
}

func TestResolveCommand(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		// "echo" should be in PATH on all platforms
		name, args, usingDir, err := resolveCommand([]string{"echo", "hello"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name == "" {
			t.Error("command name should not be empty")
		}
		if len(args) != 1 || args[0] != "hello" {
			t.Errorf("args = %v, want [hello]", args)
		}
		if usingDir {
			t.Error("usingScriptDir should be false")
		}
	})

	t.Run("nonexistent command", func(t *testing.T) {
		_, _, _, err := resolveCommand([]string{"nonexistent_wsd_command_xyz"}, "")
		if err == nil {
			t.Error("expected error for nonexistent command")
		}
	})

	t.Run("command with scriptdir is ambiguous", func(t *testing.T) {
		_, _, _, err := resolveCommand([]string{"echo"}, "/some/dir")
		if err == nil {
			t.Error("expected error for ambiguous command + dir")
		}
	})

	t.Run("no args returns empty", func(t *testing.T) {
		name, _, _, err := resolveCommand(nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "" {
			t.Errorf("expected empty command, got %q", name)
		}
	})
}

func TestResolveScriptDir(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		dir, err := resolveScriptDir("")
		if err != nil || dir != "" {
			t.Errorf("expected empty result, got %q, %v", dir, err)
		}
	})

	t.Run("valid directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		dir, err := resolveScriptDir(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		abs, _ := filepath.Abs(tmpDir)
		if dir != abs {
			t.Errorf("dir = %q, want %q", dir, abs)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := resolveScriptDir("/nonexistent/wsd/dir")
		if err == nil {
			t.Error("expected error for nonexistent dir")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(f, []byte("test"), 0644)
		_, err := resolveScriptDir(f)
		if err == nil {
			t.Error("expected error for file (not dir)")
		}
	})
}

func TestValidateDir(t *testing.T) {
	t.Run("empty dir is ok", func(t *testing.T) {
		if err := validateDir("", "test"); err != nil {
			t.Errorf("empty dir should be ok: %v", err)
		}
	})

	t.Run("valid directory", func(t *testing.T) {
		if err := validateDir(t.TempDir(), "test"); err != nil {
			t.Errorf("valid dir should be ok: %v", err)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		if err := validateDir("/nonexistent/wsd/dir", "CGI dir"); err == nil {
			t.Error("expected error for nonexistent dir")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(f, []byte("test"), 0644)
		if err := validateDir(f, "static dir"); err == nil {
			t.Error("expected error for file (not dir)")
		}
	})
}
