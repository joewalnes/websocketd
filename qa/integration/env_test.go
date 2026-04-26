package integration

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestENV001_StandardCGIVariables(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")

	required := []string{
		"SERVER_SOFTWARE",
		"GATEWAY_INTERFACE",
		"SERVER_PROTOCOL",
		"SERVER_NAME",
		"SERVER_PORT",
		"REQUEST_METHOD",
		"REMOTE_ADDR",
		"REMOTE_HOST",
	}

	for _, v := range required {
		if _, ok := findEnvValue(output, v); !ok {
			t.Errorf("required CGI variable %s not found", v)
		}
	}
}

func TestENV002_GatewayInterface(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "GATEWAY_INTERFACE"); !ok || v != "websocketd-CGI/0.1" {
		t.Errorf("GATEWAY_INTERFACE: expected 'websocketd-CGI/0.1', got %q", v)
	}
}

func TestENV003_ServerProtocol(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "SERVER_PROTOCOL"); !ok || !strings.HasPrefix(v, "HTTP/") {
		t.Errorf("SERVER_PROTOCOL: expected 'HTTP/...', got %q", v)
	}
}

func TestENV004_RequestMethod(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "REQUEST_METHOD"); !ok || v != "GET" {
		t.Errorf("REQUEST_METHOD: expected 'GET', got %q", v)
	}
}

func TestENV005_QueryString(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	// With query string
	ws := s.Connect("/?key=value&foo=bar")
	defer ws.Close()
	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "QUERY_STRING"); !ok || v != "key=value&foo=bar" {
		t.Errorf("QUERY_STRING: expected 'key=value&foo=bar', got %q", v)
	}
}

func TestENV006_QueryStringEmpty(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	ws := s.Connect("/")
	defer ws.Close()
	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	// QUERY_STRING should be present but empty
	if v, ok := findEnvValue(output, "QUERY_STRING"); ok && v != "" {
		t.Errorf("QUERY_STRING should be empty, got %q", v)
	}
}

func TestENV007_RemoteAddrAndPort(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")

	addr, ok := findEnvValue(output, "REMOTE_ADDR")
	if !ok {
		t.Fatal("REMOTE_ADDR not found")
	}
	if addr != "127.0.0.1" {
		t.Errorf("REMOTE_ADDR: expected '127.0.0.1', got %q", addr)
	}

	portStr, ok := findEnvValue(output, "REMOTE_PORT")
	if !ok {
		t.Fatal("REMOTE_PORT not found")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		t.Errorf("REMOTE_PORT: not a valid port: %q", portStr)
	}
}

func TestENV008_UniqueID(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	// Two connections should have different UNIQUE_IDs
	ws1 := s.Connect("/")
	output1 := strings.Join(collectMessages(ws1, 3*time.Second), "\n")
	id1, _ := findEnvValue(output1, "UNIQUE_ID")

	ws2 := s.Connect("/")
	output2 := strings.Join(collectMessages(ws2, 3*time.Second), "\n")
	id2, _ := findEnvValue(output2, "UNIQUE_ID")

	if id1 == "" {
		t.Error("UNIQUE_ID not set for first connection")
	}
	if id1 == id2 {
		t.Errorf("UNIQUE_ID should be unique: both got %q", id1)
	}
}

func TestENV009_RequestURI(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/path?query=1")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "REQUEST_URI"); !ok || v != "/path?query=1" {
		t.Errorf("REQUEST_URI: expected '/path?query=1', got %q", v)
	}
}

func TestENV010_HTTPHeaderConversion(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")

	headers := make(map[string][]string)
	headers["X-Custom-Test"] = []string{"custom-value"}
	headers["Accept-Language"] = []string{"en-US"}

	ws, _, err := s.TryConnect("/", headers)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")

	if v, ok := findEnvValue(output, "HTTP_X_CUSTOM_TEST"); !ok || v != "custom-value" {
		t.Errorf("HTTP_X_CUSTOM_TEST: expected 'custom-value', got %q (found: %v)", v, ok)
	}
}

func TestENV011_ServerSoftware(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "SERVER_SOFTWARE"); !ok || !strings.HasPrefix(v, "websocketd/") {
		t.Errorf("SERVER_SOFTWARE: expected 'websocketd/...', got %q", v)
	}
}

func TestENV012_ServerPort(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	expected := fmt.Sprintf("%d", s.Port)
	if v, ok := findEnvValue(output, "SERVER_PORT"); !ok || v != expected {
		t.Errorf("SERVER_PORT: expected %q, got %q", expected, v)
	}
}

func TestENV013_HTTPSVariable(t *testing.T) {
	t.Parallel()
	s := startServerSSL(t, nil, "env")
	ws := s.ConnectTLS("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "HTTPS"); !ok || v != "on" {
		t.Errorf("HTTPS: expected 'on', got %q (found: %v)", v, ok)
	}
}

func TestENV014_HTTPSNotSetWithoutSSL(t *testing.T) {
	t.Parallel()
	s := startServer(t, "env")
	ws := s.Connect("/")
	defer ws.Close()

	output := strings.Join(collectMessages(ws, 3*time.Second), "\n")
	if v, ok := findEnvValue(output, "HTTPS"); ok && v == "on" {
		t.Error("HTTPS should not be 'on' without SSL")
	}
}
