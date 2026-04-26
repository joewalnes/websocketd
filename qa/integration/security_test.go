package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSEC001_SameOriginAccepted(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--sameorigin"}, "echo")

	headers := http.Header{}
	headers.Set("Origin", "http://127.0.0.1:"+itoa(s.Port))
	ws, _, err := s.TryConnect("/", headers)
	if err != nil {
		t.Fatalf("same-origin connection should succeed: %v", err)
	}
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestSEC002_SameOriginRejected(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--sameorigin"}, "echo")

	headers := http.Header{}
	headers.Set("Origin", "http://evil.com")
	_, resp, err := s.TryConnect("/", headers)
	if err == nil {
		t.Fatal("cross-origin connection should be rejected")
	}
	if resp != nil && resp.StatusCode != 403 {
		t.Logf("rejection status: %d (expected 403)", resp.StatusCode)
	}
}

func TestSEC003_OriginWhitelistAccepted(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--origin=trusted.com"}, "echo")

	headers := http.Header{}
	headers.Set("Origin", "http://trusted.com")
	ws, _, err := s.TryConnect("/", headers)
	if err != nil {
		t.Fatalf("whitelisted origin should be accepted: %v", err)
	}
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestSEC004_OriginWhitelistRejected(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--origin=trusted.com"}, "echo")

	headers := http.Header{}
	headers.Set("Origin", "http://untrusted.com")
	_, _, err := s.TryConnect("/", headers)
	if err == nil {
		t.Fatal("non-whitelisted origin should be rejected")
	}
}

func TestSEC005_OriginWhitelistWithPort(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--origin=trusted.com:3000"}, "echo")

	// Correct port should work
	headers := http.Header{}
	headers.Set("Origin", "http://trusted.com:3000")
	ws, _, err := s.TryConnect("/", headers)
	if err != nil {
		t.Fatalf("correct port origin should be accepted: %v", err)
	}
	ws.Close()

	// Wrong port should be rejected
	headers = http.Header{}
	headers.Set("Origin", "http://trusted.com:4000")
	_, _, err = s.TryConnect("/", headers)
	if err == nil {
		t.Error("wrong port origin should be rejected")
	}
}

func TestSEC006_MultipleOrigins(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--origin=a.com,b.com"}, "echo")

	for _, origin := range []string{"http://a.com", "http://b.com"} {
		headers := http.Header{}
		headers.Set("Origin", origin)
		ws, _, err := s.TryConnect("/", headers)
		if err != nil {
			t.Errorf("origin %s should be accepted: %v", origin, err)
			continue
		}
		ws.Close()
	}

	// Unlisted origin should be rejected
	headers := http.Header{}
	headers.Set("Origin", "http://c.com")
	_, _, err := s.TryConnect("/", headers)
	if err == nil {
		t.Error("unlisted origin should be rejected")
	}
}

func TestSEC007_NullOrigin(t *testing.T) {
	// Regression: v0.2.10 fixed null origin handling
	t.Parallel()
	s := startServerOpts(t, []string{"--sameorigin"}, "echo")

	headers := http.Header{}
	headers.Set("Origin", "null")
	_, _, err := s.TryConnect("/", headers)
	if err == nil {
		t.Error("null origin should be rejected with --sameorigin")
	}
}

func TestSEC008_NoOriginRestrictionDefault(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")

	// Any origin should work when no restriction is set
	headers := http.Header{}
	headers.Set("Origin", "http://any-domain.com")
	ws, _, err := s.TryConnect("/", headers)
	if err != nil {
		t.Fatalf("should accept any origin by default: %v", err)
	}
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestSEC009_EnvironmentIsolation(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")

	// Standard CGI variables should be present
	if _, ok := findEnvValue(output, "SERVER_SOFTWARE"); !ok {
		t.Error("SERVER_SOFTWARE not found")
	}
	if _, ok := findEnvValue(output, "REMOTE_ADDR"); !ok {
		t.Error("REMOTE_ADDR not found")
	}

	// Parent shell variables should NOT be leaked
	sensitiveVars := []string{"HOME", "USER", "SHELL", "TERM", "LANG"}
	for _, v := range sensitiveVars {
		if _, ok := findEnvValue(output, v); ok {
			t.Errorf("parent variable %s leaked to child (env isolation failure)", v)
		}
	}
}

func TestSEC010_SSLConnection(t *testing.T) {
	t.Parallel()
	s := startServerSSL(t, nil, "echo")
	ws := s.ConnectTLS("/")
	defer ws.Close()
	ws.Send("encrypted")
	ws.ExpectMessage("encrypted")
}

func TestSEC011_CommandInjectionViaURL(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	// These should not execute any shell commands
	dangerousPaths := []string{
		"/;ls",
		"/$(whoami)",
		"/`id`",
		"/|cat",
	}
	for _, path := range dangerousPaths {
		ws, _, err := s.TryConnect(path, nil)
		if err != nil {
			continue // rejected is fine
		}
		// If connected, just verify no command injection happened
		output := strings.Join(collectMessages(ws, 2*time.Second), "\n")
		if strings.Contains(output, "uid=") || strings.Contains(output, "root") {
			t.Errorf("SECURITY: possible command injection via path %q", path)
		}
	}
}

func TestSEC012_CommandInjectionViaQueryString(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	ws := s.Connect("/?$(whoami)")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 2*time.Second), "\n")
	// QUERY_STRING should contain the raw text, not executed
	if v, ok := findEnvValue(output, "QUERY_STRING"); ok {
		if !strings.Contains(v, "$(whoami)") {
			t.Errorf("QUERY_STRING should contain raw text, got %q", v)
		}
	}
}

// Helper
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
