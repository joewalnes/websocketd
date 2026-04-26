package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// These tests assert the EXPECTED (correct) behavior for known bugs.
// They fail against the current code, confirming the bugs exist.
// Each is skipped with an explanation after being verified to fail.

func TestBUG001_HeaderShouldApplyToAllResponses(t *testing.T) {
	t.Skip("BUG-001: --header only applies to WebSocket responses, not HTTP. See qa/bugs-found.md.")

	t.Parallel()
	s := startServerOpts(t, []string{"--header=X-Custom: value"}, "echo")

	// --header should apply to ALL responses, including plain HTTP
	resp, _ := s.HTTPGet("/")
	if v := resp.Header.Get("X-Custom"); v != "value" {
		t.Errorf("--header not present in HTTP response: got %q", v)
	}

	// And also WebSocket
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp2, err := dialer.Dial(s.WSURL("/"), nil)
	if err != nil {
		t.Fatalf("ws connect failed: %v", err)
	}
	defer conn.Close()
	if v := resp2.Header.Get("X-Custom"); v != "value" {
		t.Errorf("--header not present in WebSocket response: got %q", v)
	}
}

func TestBUG002_GatewayInterfaceShouldBeStandard(t *testing.T) {
	t.Skip("BUG-002: GATEWAY_INTERFACE is 'websocketd-CGI/0.1', not RFC 3875 'CGI/1.1'. See qa/bugs-found.md.")

	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "GATEWAY_INTERFACE"); !ok || v != "CGI/1.1" {
		t.Errorf("GATEWAY_INTERFACE should be 'CGI/1.1' per RFC 3875, got %q", v)
	}
}

func TestBUG004_GorillaDependencyVersion(t *testing.T) {
	t.Skip("BUG-004: gorilla/websocket v1.4.0 is outdated and archived. See qa/bugs-found.md.")

	// This is an audit finding, not a runtime test.
	// gorilla/websocket v1.4.0 was released in 2018.
	// The library has been archived by maintainers.
	// Run: govulncheck ./...
	t.Fatal("gorilla/websocket dependency needs updating")
}

func TestBUG005_GoModuleVersion(t *testing.T) {
	t.Skip("BUG-005: go.mod specifies Go 1.15 which is end-of-life. See qa/bugs-found.md.")

	// go.mod has: go 1.15
	// Go 1.15 reached end of life in 2021.
	t.Fatal("go.mod should specify a supported Go version (1.21+)")
}

func TestBUG006_LargeBinaryPayloadStalls(t *testing.T) {
	t.Skip("BUG-006: Binary payloads >64KB stall due to pipe buffer limits. See qa/bugs-found.md.")

	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()

	// 1MB binary payload should round-trip successfully
	size := 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	ws.SendBinary(data)

	ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, recv, err := ws.conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to receive 1MB binary: %v", err)
	}
	if len(recv) != size {
		t.Errorf("expected %d bytes, got %d", size, len(recv))
	}
}

func TestBUG001_HeaderHTTPShouldNotAffectWebSocket(t *testing.T) {
	t.Skip("BUG-001 (corollary): No way to set headers on ALL responses with a single flag. See qa/bugs-found.md.")

	t.Parallel()
	// There should be a single flag that sets headers on both HTTP and WS responses.
	// Currently you need both --header (WS only) and --header-http (HTTP only).
	s := startServerOpts(t, []string{"--header=X-Universal: yes"}, "echo")

	// Check HTTP
	resp, _ := s.HTTPGet("/")
	httpVal := resp.Header.Get("X-Universal")

	// Check WS
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp2, err := dialer.Dial(s.WSURL("/"), nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()
	wsVal := resp2.Header.Get("X-Universal")

	if httpVal != "yes" || wsVal != "yes" {
		t.Errorf("--header should be universal: HTTP=%q, WS=%q", httpVal, wsVal)
	}
}

func TestBUG003_StaticFileShouldNotRedirect(t *testing.T) {
	t.Skip("BUG-003: Static file serving may 301 redirect due to Go http.FileServer path canonicalization. See qa/bugs-found.md.")

	t.Parallel()
	dir := t.TempDir()
	writeFile(dir, "index.html", "<html>hello</html>")

	s := startServerOpts(t, []string{"--staticdir=" + dir}, "echo")

	// A direct request to a known file should return 200, not 301
	resp, _ := s.HTTPGet("/index.html")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /index.html, got %d (redirect to %s)",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
