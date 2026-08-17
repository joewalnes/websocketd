package integration

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// rawHTTPGet sends a request line verbatim, without any client-side URL
// normalization, and returns the response. Go's http.Client (like curl)
// rewrites dot segments before sending, which hides exactly the traversal
// these tests are about.
func rawHTTPGet(t *testing.T, port int, requestTarget string) *http.Response {
	t.Helper()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	req := "GET " + requestTarget + " HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write request: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response for %q: %v", requestTarget, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(body)
}

// writeCGIScript writes an executable CGI script printing the given body.
func writeCGIScript(t *testing.T, path, body string) {
	t.Helper()
	script := "#!/bin/sh\nprintf 'Content-Type: text/plain\\r\\n\\r\\n'\nprintf '%s\\n'\n"
	if err := os.WriteFile(path, []byte(fmt.Sprintf(script, body)), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCGI001_ScriptExecuted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("CGI scripts here are /bin/sh")
	}
	t.Parallel()
	cgiDir := t.TempDir()
	writeCGIScript(t, filepath.Join(cgiDir, "hello.sh"), "hello-from-cgi")
	if err := os.Mkdir(filepath.Join(cgiDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	writeCGIScript(t, filepath.Join(cgiDir, "sub", "nested.sh"), "hello-from-nested")

	s := startServerOpts(t, []string{"--cgidir=" + cgiDir}, "echo")

	resp, body := s.HTTPGet("/hello.sh")
	if resp.StatusCode != 200 || !strings.Contains(body, "hello-from-cgi") {
		t.Errorf("expected CGI script to run, got %d %q", resp.StatusCode, body)
	}

	resp, body = s.HTTPGet("/sub/nested.sh")
	if resp.StatusCode != 200 || !strings.Contains(body, "hello-from-nested") {
		t.Errorf("expected nested CGI script to run, got %d %q", resp.StatusCode, body)
	}

	// Dot segments that stay inside the directory still resolve (the client
	// follows the mux's normalization redirect).
	resp, body = s.HTTPGetFollow("/sub/../hello.sh")
	if resp.StatusCode != 200 || !strings.Contains(body, "hello-from-cgi") {
		t.Errorf("expected interior dot segment to resolve, got %d %q", resp.StatusCode, body)
	}

	resp, _ = s.HTTPGet("/nosuch.sh")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for missing script, got %d", resp.StatusCode)
	}
}

func rawResolve(t *testing.T, s *Server, target string) (*http.Response, string) {
	t.Helper()
	resp := rawHTTPGet(t, s.Port, target)
	return resp, readBody(t, resp)
}

// TestCGI002_PathTraversal is the regression test for the CGI directory
// escape: nothing outside --cgidir may be executed, however the request
// path is spelled.
func TestCGI002_PathTraversal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("CGI scripts here are /bin/sh")
	}
	t.Parallel()

	// An executable sitting outside the CGI directory. It is a sibling of
	// cgiDir so a short "../" hop reaches it.
	base := t.TempDir()
	cgiDir := filepath.Join(base, "cgi")
	if err := os.Mkdir(cgiDir, 0755); err != nil {
		t.Fatal(err)
	}
	outsideDir := filepath.Join(base, "outside")
	if err := os.Mkdir(outsideDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeCGIScript(t, filepath.Join(outsideDir, "evil.sh"), "PWNED")
	writeCGIScript(t, filepath.Join(cgiDir, "ok.sh"), "ok")

	s := startServerOpts(t, []string{"--cgidir=" + cgiDir}, "echo")

	targets := []string{
		"/../outside/evil.sh",
		"/%2e%2e/outside/evil.sh",
		"/.%2e/outside/evil.sh",
		"/ok.sh/../../outside/evil.sh",
		"/sub/../../outside/evil.sh",
		"/..%2Foutside%2Fevil.sh",
	}
	for _, target := range targets {
		resp, body := rawResolve(t, s, target)
		if strings.Contains(body, "PWNED") {
			t.Errorf("SECURITY: %q escaped the CGI directory and executed (status %d)", target, resp.StatusCode)
		}
		if resp.StatusCode == 200 {
			t.Errorf("SECURITY: %q returned 200, expected refusal", target)
		}
	}

	// A symlink inside the CGI directory pointing outside it must not be
	// executed either.
	if err := os.Symlink(filepath.Join(outsideDir, "evil.sh"), filepath.Join(cgiDir, "escape.sh")); err != nil {
		t.Fatal(err)
	}
	resp, body := rawResolve(t, s, "/escape.sh")
	if strings.Contains(body, "PWNED") {
		t.Errorf("SECURITY: symlink out of the CGI directory executed (status %d)", resp.StatusCode)
	}

	// The legitimate script is unaffected.
	resp, body = rawResolve(t, s, "/ok.sh")
	if resp.StatusCode != 200 || !strings.Contains(body, "ok") {
		t.Errorf("expected the in-directory script to still run, got %d %q", resp.StatusCode, body)
	}
}
