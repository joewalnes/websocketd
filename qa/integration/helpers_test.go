package integration

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketdBin string
	testcmdBin    string
)

func TestMain(m *testing.M) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find project root: %v\n", err)
		os.Exit(1)
	}

	tmpDir, err := os.MkdirTemp("", "wstest-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	websocketdBin = filepath.Join(tmpDir, "websocketd"+ext)
	cmd := exec.Command("go", "build", "-o", websocketdBin, ".")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build websocketd: %v\n%s\n", err, out)
		os.Exit(1)
	}

	testcmdBin = filepath.Join(tmpDir, "testcmd"+ext)
	cmd = exec.Command("go", "build", "-o", testcmdBin, "./qa/integration/testcmd")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build testcmd: %v\n%s\n", err, out)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// syncBuffer is a goroutine-safe bytes.Buffer for capturing stderr.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Server wraps a running websocketd instance for testing.
type Server struct {
	t          *testing.T
	Port       int
	IsHTTPS    bool
	cmd        *exec.Cmd
	stderr     syncBuffer
	stderrDone chan struct{}
}

// startServer starts websocketd with testcmd <mode> as the backend.
func startServer(t *testing.T, mode string, modeArgs ...string) *Server {
	return startServerOpts(t, nil, mode, modeArgs...)
}

// startServerOpts starts websocketd with extra flags and testcmd <mode>.
func startServerOpts(t *testing.T, wsFlags []string, mode string, modeArgs ...string) *Server {
	cmdArgs := append([]string{mode}, modeArgs...)
	return startServerRaw(t, wsFlags, testcmdBin, cmdArgs...)
}

// startServerRaw starts websocketd with arbitrary flags and command.
func startServerRaw(t *testing.T, wsFlags []string, command string, cmdArgs ...string) *Server {
	t.Helper()
	port := freePort(t)

	args := []string{
		"--port=" + strconv.Itoa(port),
		"--address=127.0.0.1",
		"--loglevel=access",
	}
	args = append(args, wsFlags...)
	args = append(args, command)
	args = append(args, cmdArgs...)

	cmd := exec.Command(websocketdBin, args...)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	s := &Server{
		t:          t,
		Port:       port,
		cmd:        cmd,
		stderrDone: make(chan struct{}),
	}

	go func() {
		io.Copy(&s.stderr, stderrPipe)
		close(s.stderrDone)
	}()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start websocketd: %v", err)
	}

	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
		<-s.stderrDone
		if t.Failed() {
			t.Logf("websocketd stderr:\n%s", s.stderr.String())
		}
	})

	waitForPort(t, port, 10*time.Second)
	return s
}

// startServerSSL starts websocketd with TLS using a generated self-signed cert.
func startServerSSL(t *testing.T, extraFlags []string, mode string, modeArgs ...string) *Server {
	t.Helper()
	certFile, keyFile := generateTestCert(t)
	flags := []string{
		"--ssl",
		"--sslcert=" + certFile,
		"--sslkey=" + keyFile,
	}
	flags = append(flags, extraFlags...)
	s := startServerOpts(t, flags, mode, modeArgs...)
	s.IsHTTPS = true
	return s
}

// Connect opens a WebSocket connection to the server or fails the test.
func (s *Server) Connect(path string) *WSClient {
	s.t.Helper()
	ws, _, err := s.TryConnect(path, nil)
	if err != nil {
		s.t.Fatalf("failed to connect to ws://127.0.0.1:%d%s: %v", s.Port, path, err)
	}
	return ws
}

// ConnectTLS opens a secure WebSocket connection.
func (s *Server) ConnectTLS(path string) *WSClient {
	s.t.Helper()
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}
	url := fmt.Sprintf("wss://127.0.0.1:%d%s", s.Port, path)
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		s.t.Fatalf("failed to connect to %s: %v", url, err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	ws := &WSClient{t: s.t, conn: conn}
	s.t.Cleanup(func() { conn.Close() })
	return ws
}

// TryConnect attempts a WebSocket connection, returning error instead of failing.
func (s *Server) TryConnect(path string, headers http.Header) (*WSClient, *http.Response, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	scheme := "ws"
	if s.IsHTTPS {
		scheme = "wss"
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	url := fmt.Sprintf("%s://127.0.0.1:%d%s", scheme, s.Port, path)
	conn, resp, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, resp, err
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	ws := &WSClient{t: s.t, conn: conn}
	s.t.Cleanup(func() { conn.Close() })
	return ws, resp, nil
}

// WSURL returns the WebSocket URL for the given path.
func (s *Server) WSURL(path string) string {
	scheme := "ws"
	if s.IsHTTPS {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://127.0.0.1:%d%s", scheme, s.Port, path)
}

// HTTPURL returns the HTTP URL for the given path.
func (s *Server) HTTPURL(path string) string {
	scheme := "http"
	if s.IsHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://127.0.0.1:%d%s", scheme, s.Port, path)
}

// HTTPGet makes an HTTP GET request and returns response + body.
func (s *Server) HTTPGet(path string) (*http.Response, string) {
	s.t.Helper()
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if s.IsHTTPS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Get(s.HTTPURL(path))
	if err != nil {
		s.t.Fatalf("HTTP GET %s failed: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

// Stderr returns the captured stderr output from websocketd.
func (s *Server) Stderr() string {
	return s.stderr.String()
}

// HTTPGetFollow makes an HTTP GET that follows redirects and returns response + body.
func (s *Server) HTTPGetFollow(path string) (*http.Response, string) {
	s.t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	if s.IsHTTPS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Get(s.HTTPURL(path))
	if err != nil {
		s.t.Fatalf("HTTP GET %s failed: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

// WSClient wraps a WebSocket connection for testing.
type WSClient struct {
	t    *testing.T
	conn *websocket.Conn
}

// Send sends a text message.
func (c *WSClient) Send(msg string) {
	c.t.Helper()
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		c.t.Fatalf("failed to send message: %v", err)
	}
}

// SendBinary sends a binary message.
func (c *WSClient) SendBinary(data []byte) {
	c.t.Helper()
	if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		c.t.Fatalf("failed to send binary message: %v", err)
	}
}

// Recv receives a text message or fails.
func (c *WSClient) Recv() string {
	c.t.Helper()
	msg, err := c.RecvTimeout(5 * time.Second)
	if err != nil {
		c.t.Fatalf("failed to receive message: %v", err)
	}
	return msg
}

// RecvTimeout receives a text message with a custom timeout.
func (c *WSClient) RecvTimeout(timeout time.Duration) (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer c.conn.SetReadDeadline(time.Time{})
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		return "", err
	}
	return string(msg), nil
}

// RecvBinary receives a binary message.
func (c *WSClient) RecvBinary() (int, []byte) {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})
	msgType, msg, err := c.conn.ReadMessage()
	if err != nil {
		c.t.Fatalf("failed to receive binary message: %v", err)
	}
	return msgType, msg
}

// ExpectMessage receives a message and asserts it matches expected.
func (c *WSClient) ExpectMessage(expected string) {
	c.t.Helper()
	got := c.Recv()
	if got != expected {
		c.t.Errorf("expected message %q, got %q", expected, got)
	}
}

// ExpectMessages receives multiple messages in order.
func (c *WSClient) ExpectMessages(expected ...string) {
	c.t.Helper()
	for _, exp := range expected {
		c.ExpectMessage(exp)
	}
}

// ExpectClosed asserts the connection is closed.
func (c *WSClient) ExpectClosed() {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := c.conn.ReadMessage()
	if err == nil {
		c.t.Error("expected connection to be closed, but received a message")
	}
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() {
	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.conn.Close()
}

// freePort finds an available TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// waitForPort polls until a TCP connection succeeds on the given port.
func waitForPort(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start on port %d within %v", port, timeout)
}

// runWebsocketd runs websocketd with args and returns output and exit code.
// Used for testing flags that cause immediate exit (--version, --help, errors).
func runWebsocketd(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(websocketdBin, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run websocketd: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// generateTestCert creates a self-signed TLS certificate for testing.
func generateTestCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certFile, _ := os.Create(certPath)
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyFile, _ := os.Create(keyPath)
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	return
}

// writeFile writes content to a file in the given directory.
func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

// findEnvValue finds the value of an environment variable in env dump output.
func findEnvValue(envOutput, varName string) (string, bool) {
	prefix := varName + "="
	for _, line := range strings.Split(envOutput, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), true
		}
	}
	return "", false
}

// retryConnect polls until a new WebSocket connection succeeds and receives
// a message, or fails the test after timeout. Use this instead of time.Sleep
// when verifying that the server is still alive after a disconnect.
func (s *Server) retryConnect(t *testing.T, path string, timeout time.Duration) *WSClient {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ws, _, err := s.TryConnect(path, nil)
		if err == nil {
			return ws
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("could not reconnect to server within %v", timeout)
	return nil
}

// collectMessages reads all messages until the connection closes or timeout.
func collectMessages(ws *WSClient, timeout time.Duration) []string {
	var msgs []string
	for {
		msg, err := ws.RecvTimeout(timeout)
		if err != nil {
			break
		}
		msgs = append(msgs, msg)
	}
	return msgs
}
