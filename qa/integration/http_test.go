package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTTP001_StaticFileServing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>hello</html>"), 0644)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte("body { color: red; }"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "page.html"), []byte("<html>sub</html>"), 0644)

	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	// Serve HTML (follow redirects in case FileServer canonicalizes paths)
	resp, body := s.HTTPGetFollow("/index.html")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for index.html, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "hello") {
		t.Errorf("index.html body: %q", body)
	}

	// Serve CSS
	resp, body = s.HTTPGetFollow("/style.css")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for style.css, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "color") {
		t.Errorf("style.css body: %q", body)
	}

	// Serve subdirectory
	resp, body = s.HTTPGetFollow("/sub/page.html")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for sub/page.html, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "sub") {
		t.Errorf("sub/page.html body: %q", body)
	}
}

func TestHTTP002_StaticFile404(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.html"), []byte("here"), 0644)

	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	resp, _ := s.HTTPGet("/nonexistent.html")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHTTP003_StaticFilePathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "safe.html"), []byte("safe"), 0644)

	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	// Try path traversal
	resp, body := s.HTTPGet("/../../../etc/passwd")
	if resp.StatusCode == 200 && strings.Contains(body, "root:") {
		t.Error("SECURITY: path traversal succeeded — /etc/passwd served")
	}

	resp, body = s.HTTPGet("/..%2F..%2F..%2Fetc%2Fpasswd")
	if resp.StatusCode == 200 && strings.Contains(body, "root:") {
		t.Error("SECURITY: encoded path traversal succeeded")
	}
}

func TestHTTP004_StaticAndWebSocketCoexist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "page.html"), []byte("<html>page</html>"), 0644)

	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	// Static file works
	resp, body := s.HTTPGet("/page.html")
	if resp.StatusCode != 200 || !strings.Contains(body, "page") {
		t.Error("static file serving failed")
	}

	// WebSocket still works
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestHTTP005_DevConsoleServing(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--devconsole"}, "echo")

	resp, body := s.HTTPGet("/")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "html") {
		t.Error("dev console should return HTML")
	}
}

func TestHTTP006_DevConsoleAndWebSocket(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--devconsole"}, "echo")

	// Dev console serves HTML
	resp, _ := s.HTTPGet("/")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// WebSocket still works
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestHTTP007_CustomHeadersOnHTTP(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.html"), []byte("test"), 0644)

	s := startServerOpts(t,
		[]string{
			"--staticdir=" + dir,
			"--header-http=X-HTTP: only-http",
		},
		"echo")

	resp, _ := s.HTTPGetFollow("/test.html")
	if v := resp.Header.Get("X-HTTP"); v != "only-http" {
		t.Errorf("X-HTTP not in HTTP response: %q", v)
	}
}

func TestHTTP008_QueryStringPassedToScript(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/?foo=bar&baz=qux")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 2*time.Second), "\n")
	if v, ok := findEnvValue(output, "QUERY_STRING"); !ok || v != "foo=bar&baz=qux" {
		t.Errorf("QUERY_STRING expected 'foo=bar&baz=qux', got %q (found: %v)", v, ok)
	}
}

func TestHTTP009_MultipleURLPaths(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	// Different paths should all route to the same script in single-command mode
	for _, path := range []string{"/", "/path", "/deep/nested/path"} {
		ws := s.Connect(path)
		output := strings.Join(collectMessages(ws, 2*time.Second), "\n")
		if _, ok := findEnvValue(output, "SERVER_SOFTWARE"); !ok {
			t.Errorf("path %s: SERVER_SOFTWARE not found in env output", path)
		}
	}
}
