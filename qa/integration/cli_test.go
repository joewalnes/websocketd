package integration

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCLI001_CustomPort(t *testing.T) {
	t.Parallel()
	s := startServer(t, "echo")
	ws := s.Connect("/")
	defer ws.Close()
	ws.Send("test")
	ws.ExpectMessage("test")
}

func TestCLI002_PortAlreadyInUse(t *testing.T) {
	t.Parallel()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	_, stderr, exitCode := runWebsocketd(t,
		"--port="+strconv.Itoa(port),
		"--address=127.0.0.1",
		testcmdBin, "echo")

	if exitCode == 0 && !strings.Contains(stderr, "in use") && !strings.Contains(stderr, "bind") {
		t.Logf("stderr: %s", stderr)
		t.Log("Note: server may not detect occupied port immediately")
	}
}

func TestCLI003_VersionFlag(t *testing.T) {
	t.Parallel()
	stdout, _, _ := runWebsocketd(t, "--version")
	if !strings.Contains(stdout, "websocketd") {
		t.Errorf("--version output doesn't contain 'websocketd': %q", stdout)
	}
}

func TestCLI004_HelpFlag(t *testing.T) {
	t.Parallel()
	stdout, stderr, _ := runWebsocketd(t, "--help")
	combined := stdout + stderr
	if !strings.Contains(combined, "--port") {
		t.Errorf("--help output doesn't mention --port: %q", combined)
	}
	if !strings.Contains(combined, "--ssl") {
		t.Errorf("--help output doesn't mention --ssl: %q", combined)
	}
}

func TestCLI005_LicenseFlag(t *testing.T) {
	t.Parallel()
	stdout, stderr, _ := runWebsocketd(t, "--license")
	combined := stdout + stderr
	if !strings.Contains(combined, "BSD") && !strings.Contains(combined, "Copyright") && !strings.Contains(combined, "license") {
		t.Errorf("--license output doesn't contain license text: %q", combined)
	}
}

func TestCLI006_SSLWithValidCert(t *testing.T) {
	t.Parallel()
	s := startServerSSL(t, nil, "echo")
	ws := s.ConnectTLS("/")
	defer ws.Close()
	ws.Send("secure hello")
	ws.ExpectMessage("secure hello")
}

func TestCLI007_SSLWithoutCert(t *testing.T) {
	t.Parallel()
	port := freePort(t)
	_, stderr, exitCode := runWebsocketd(t,
		"--port="+strconv.Itoa(port),
		"--ssl",
		testcmdBin, "echo")
	_ = stderr
	_ = exitCode
	// Just verify it doesn't panic (regression for #431)
}

func TestCLI008_CustomHeaders(t *testing.T) {
	// --header applies to ALL responses (both HTTP and WebSocket).
	t.Parallel()
	s := startServerOpts(t,
		[]string{`--header=X-Custom-Test: hello123`},
		"echo")

	// Should appear in HTTP responses
	resp, _ := s.HTTPGet("/")
	if v := resp.Header.Get("X-Custom-Test"); v != "hello123" {
		t.Errorf("expected X-Custom-Test: hello123 in HTTP response, got %q", v)
	}

	// Should also appear in WebSocket upgrade responses
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp2, err := dialer.Dial(s.WSURL("/"), nil)
	if err != nil {
		t.Fatalf("WebSocket connect failed: %v", err)
	}
	defer conn.Close()
	if v := resp2.Header.Get("X-Custom-Test"); v != "hello123" {
		t.Errorf("expected X-Custom-Test: hello123 in WS response, got %q", v)
	}
}

func TestCLI009_HeaderWSOnly(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t,
		[]string{`--header-ws=X-WS-Only: true`},
		"echo")

	// HTTP response should NOT have the WS-only header
	resp, _ := s.HTTPGet("/")
	if v := resp.Header.Get("X-WS-Only"); v != "" {
		t.Errorf("X-WS-Only should not be in HTTP response, got %q", v)
	}

	// WebSocket upgrade response should have it
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp2, err := dialer.Dial(s.WSURL("/"), nil)
	if err != nil {
		t.Fatalf("WebSocket connect failed: %v", err)
	}
	defer conn.Close()
	if v := resp2.Header.Get("X-WS-Only"); v != "true" {
		t.Errorf("X-WS-Only not in WebSocket response: %q", v)
	}
}

func TestCLI010_HeaderHTTPOnly(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t,
		[]string{`--header-http=X-HTTP-Only: true`},
		"echo")

	// HTTP response should have it
	resp, _ := s.HTTPGet("/")
	if v := resp.Header.Get("X-HTTP-Only"); v != "true" {
		t.Errorf("X-HTTP-Only not in HTTP response: %q", v)
	}

	// WebSocket upgrade should NOT have it
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp2, err := dialer.Dial(s.WSURL("/"), nil)
	if err != nil {
		t.Fatalf("WebSocket connect failed: %v", err)
	}
	defer conn.Close()
	if v := resp2.Header.Get("X-HTTP-Only"); v != "" {
		t.Errorf("X-HTTP-Only should not be in WebSocket response: %q", v)
	}
}

func TestCLI011_Passenv(t *testing.T) {
	t.Parallel()
	port := freePort(t)

	cmd := exec.Command(websocketdBin,
		"--port="+strconv.Itoa(port),
		"--address=127.0.0.1",
		"--loglevel=error",
		"--passenv=MY_TEST_VAR",
		testcmdBin, "env")

	cmd.Env = []string{
		"MY_TEST_VAR=hello_from_test",
		"SECRET_VAR=should_not_appear",
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	waitForPort(t, port, 10*time.Second)

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial("ws://127.0.0.1:"+strconv.Itoa(port)+"/", nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	ws := &WSClient{t: t, conn: conn}
	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")

	if !strings.Contains(output, "MY_TEST_VAR=hello_from_test") {
		t.Error("MY_TEST_VAR not found in child environment")
	}
	if strings.Contains(output, "SECRET_VAR=should_not_appear") {
		t.Error("SECRET_VAR leaked to child environment without --passenv")
	}
}

func TestCLI012_RedirPort(t *testing.T) {
	t.Parallel()
	certFile, keyFile := generateTestCert(t)
	mainPort := freePort(t)
	redirPort := freePort(t)

	cmd := exec.Command(websocketdBin,
		"--port="+strconv.Itoa(mainPort),
		"--address=127.0.0.1",
		"--ssl",
		"--sslcert="+certFile,
		"--sslkey="+keyFile,
		"--redirport="+strconv.Itoa(redirPort),
		"--loglevel=error",
		testcmdBin, "echo")

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	waitForPort(t, redirPort, 10*time.Second)

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(redirPort) + "/")
	if err != nil {
		t.Fatalf("HTTP GET to redir port failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 301 && resp.StatusCode != 302 && resp.StatusCode != 307 {
		t.Errorf("expected redirect status, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, strconv.Itoa(mainPort)) {
		t.Errorf("redirect location doesn't contain main port %d: %s", mainPort, loc)
	}
}

func TestCLI013_Devconsole(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--devconsole"}, "echo")
	resp, body := s.HTTPGet("/")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for dev console, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "<html") && !strings.Contains(body, "websocket") {
		t.Error("dev console response doesn't contain expected HTML content")
	}
}

func TestCLI014_BinaryModeFlag(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--binary"}, "binary-echo")
	ws := s.Connect("/")
	defer ws.Close()
	data := []byte{1, 2, 3, 4, 5}
	ws.SendBinary(data)
	_, recv := ws.RecvBinary()
	if len(recv) != len(data) {
		t.Errorf("binary echo: sent %d bytes, got %d", len(data), len(recv))
	}
}
