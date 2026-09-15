//go:build !windows

package integration

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestPROC013_SessionTeardownKillsProcessGroup verifies that when a client
// disconnects, websocketd tears down the whole process group — not just the
// direct child. Previously a script could spawn a background child, exit,
// and leave that child running (and, by holding the inherited pipes, the
// session and its --maxforks slot occupied) long after the connection was
// gone, because signals were sent to the direct child only.
func TestPROC013_SessionTeardownKillsProcessGroup(t *testing.T) {
	t.Parallel()

	// The grandchild prints its pid, then the script serves stdin so the
	// session stays alive until we close it. sleep 60 makes it unmistakable
	// if teardown fails to reach it.
	script := `sleep 60 & echo GRANDCHILD=$!; while IFS= read -r line; do echo "R:$line"; done`
	s := startServerRaw(t, nil, "/bin/sh", "-c", script)

	ws := s.Connect("/")
	bg := strings.TrimPrefix(ws.Recv(), "GRANDCHILD=")
	pid, err := strconv.Atoi(strings.TrimSpace(bg))
	if err != nil {
		t.Fatalf("grandchild pid %q is not a number: %v", bg, err)
	}
	t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL) }) // don't leak it if the test fails

	ws.Send("hello")
	ws.ExpectMessage("R:hello")

	// Close the session; the grandchild must not survive it.
	ws.Close()

	// Signal(0) succeeds while the process exists (or is an unreaped zombie).
	// The grandchild's parent (the script) has exited, so it is reparented to
	// init and gets reaped quickly once killed — a short poll is enough.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if proc, err := os.FindProcess(pid); err == nil {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				return // gone — teardown reached it
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("grandchild %d still alive %v after the session closed", pid, 10*time.Second)
}
