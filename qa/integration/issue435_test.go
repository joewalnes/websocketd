package integration

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Tests for #435: serving over a Unix domain socket via --unixsocket.

// dialUnixSocket connects a WSClient to a websocketd instance listening on a
// Unix domain socket. The URL host is ignored by the custom dialer; only the
// path matters.
func dialUnixSocket(t *testing.T, sockPath, path string) *WSClient {
	t.Helper()
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", sockPath)
		},
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial("ws://unix-socket"+path, nil)
	if err != nil {
		t.Fatalf("failed to dial unix socket %s: %v", sockPath, err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	ws := &WSClient{t: t, conn: conn}
	t.Cleanup(func() { conn.Close() })
	return ws
}

// waitForSocket polls until a Unix domain socket file is dialable.
func waitForSocket(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", path); err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for unix socket %s", path)
}

func skipUnixSocketOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		// AF_UNIX support on Windows is recent and CI coverage for it is
		// thin; skip execution here rather than risk a flaky CI signal.
		// The --unixsocket flag itself is not restricted to non-Windows.
		t.Skip("skipping Unix domain socket test on windows")
	}
}

// shortSocketPath returns a socket path short enough to survive macOS's
// 104-byte sockaddr_un.sun_path limit. t.TempDir() is unusable here: on
// macOS CI, $TMPDIR is already a long random path, and nesting the test
// name and a "/001/" subdir under it routinely blows past the limit
// (bind: invalid argument). /tmp itself is always short.
func shortSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "wsd")
	if err != nil {
		t.Fatalf("failed to create temp dir for socket: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

func TestIssue435_UnixSocketEcho(t *testing.T) {
	skipUnixSocketOnWindows(t)
	t.Parallel()

	sockPath := shortSocketPath(t)
	args := []string{"--unixsocket=" + sockPath, "--loglevel=access", testcmdBin, "echo"}
	cmd := startServerRawArgs(t, args)
	waitForSocket(t, sockPath, 10*time.Second)

	ws := dialUnixSocket(t, sockPath, "/")
	ws.Send("hello over uds")
	ws.ExpectMessage("hello over uds")

	if strings.Contains(cmd.Stdout(), "ws://") {
		t.Error("expected no TCP listener log line when only --unixsocket is set")
	}
	if !strings.Contains(cmd.Stdout(), "unix socket at "+sockPath) {
		t.Errorf("expected a log line announcing the unix socket, got:\n%s", cmd.Stdout())
	}
}

func TestIssue435_StaleSocketCleanup(t *testing.T) {
	skipUnixSocketOnWindows(t)
	t.Parallel()

	sockPath := shortSocketPath(t)

	// Simulate a leftover socket file from an unclean shutdown (e.g. a
	// killed process, which never gets to run its own cleanup). A graceful
	// Close() would auto-unlink the file, so disable that explicitly to
	// leave the file behind, same as a hard kill would.
	stale, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to create stale socket: %v", err)
	}
	stale.(*net.UnixListener).SetUnlinkOnClose(false)
	stale.Close()
	if _, err := os.Stat(sockPath); err != nil {
		t.Fatalf("expected stale socket file to remain on disk: %v", err)
	}

	args := []string{"--unixsocket=" + sockPath, "--loglevel=access", testcmdBin, "echo"}
	startServerRawArgs(t, args)
	waitForSocket(t, sockPath, 10*time.Second)

	ws := dialUnixSocket(t, sockPath, "/")
	ws.Send("still works")
	ws.ExpectMessage("still works")
}
