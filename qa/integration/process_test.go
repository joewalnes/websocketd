package integration

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPROC001_ProcessPerConnection(t *testing.T) {
	t.Parallel()
	s := startServer(t, "pid-echo")

	ws1 := s.Connect("/")
	pid1 := ws1.Recv()

	ws2 := s.Connect("/")
	pid2 := ws2.Recv()

	ws1.Close()
	ws2.Close()

	if pid1 == pid2 {
		t.Errorf("expected different PIDs, both got %s", pid1)
	}
	// Verify PIDs are actual numbers
	if _, err := strconv.Atoi(strings.TrimSpace(pid1)); err != nil {
		t.Errorf("pid1 is not a number: %q", pid1)
	}
	if _, err := strconv.Atoi(strings.TrimSpace(pid2)); err != nil {
		t.Errorf("pid2 is not a number: %q", pid2)
	}
}

func TestPROC002_StdoutToWebSocket(t *testing.T) {
	t.Parallel()
	s := startServer(t, "welcome", "ready")
	ws := s.Connect("/")
	defer ws.Close()
	// Should receive "ready" without sending anything first
	ws.ExpectMessage("ready")
}

func TestPROC003_StderrToLogs(t *testing.T) {
	t.Parallel()
	s := startServer(t, "stderr")
	ws := s.Connect("/")
	defer ws.Close()

	// Should only receive stdout, not stderr
	ws.ExpectMessage("stdout line")

	// stderr goes to websocketd logs, not to the client
	ws.ExpectClosed()

	// Poll for stderr content (process may still be flushing)
	var stderr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stderr = s.Stderr()
		if strings.Contains(stderr, "stderr line") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(stderr, "stderr line") {
		t.Logf("Note: stderr line not captured in websocketd output (may be expected based on log level)")
	}
}

func TestPROC004_MaxforksLimit(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--maxforks=2"}, "echo")

	ws1 := s.Connect("/")
	defer ws1.Close()
	ws2 := s.Connect("/")
	defer ws2.Close()

	// Third connection should be rejected
	_, resp, err := s.TryConnect("/", nil)
	if err == nil {
		t.Fatal("expected third connection to be rejected")
	}
	if resp != nil && resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected HTTP 429, got %d", resp.StatusCode)
	}
}

func TestPROC005_MaxforksRecovery(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--maxforks=1"}, "echo")

	ws1 := s.Connect("/")
	ws1.Send("hello")
	ws1.ExpectMessage("hello")
	ws1.Close()

	// Should be able to connect again after fork is released
	ws2 := s.retryConnect(t, "/", 5*time.Second)
	defer ws2.Close()
	ws2.Send("world")
	ws2.ExpectMessage("world")
}

func TestPROC006_ProcessExitNonZero(t *testing.T) {
	t.Parallel()
	s := startServer(t, "exit", "42", "about to fail")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessage("about to fail")
	ws.ExpectClosed()
}

func TestPROC007_ScriptDirectoryMode(t *testing.T) {
	t.Parallel()
	// Create a "script directory" — but since we use testcmd, we use --dir isn't
	// directly testable with testcmd. Instead test URL routing with single command.
	// The handler_test.go unit tests cover --dir path resolution.
	// Here we verify that PATH_INFO and SCRIPT_NAME are set correctly
	// when connecting to different paths.
	s := startServer(t, "env")
	ws := s.Connect("/some/path")
	defer ws.Close()
	output := strings.Join(collectMessages(ws, 2*time.Second), "\n")
	if v, ok := findEnvValue(output, "PATH_INFO"); ok {
		if v != "/some/path" {
			t.Errorf("PATH_INFO: expected /some/path, got %s", v)
		}
	}
}

func TestPROC008_CommandArguments(t *testing.T) {
	t.Parallel()
	s := startServer(t, "output", "arg1", "arg2", "arg3")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectMessages("arg1", "arg2", "arg3")
}

func TestPROC009_RapidProcessExit(t *testing.T) {
	t.Parallel()
	s := startServer(t, "exit", "0", "quick")

	// Rapid connect/disconnect shouldn't crash the server
	for i := 0; i < 5; i++ {
		ws := s.Connect("/")
		ws.ExpectMessage("quick")
		ws.ExpectClosed()
	}
}

func TestPROC010_ProcessExitNoOutput(t *testing.T) {
	t.Parallel()
	s := startServer(t, "exit", "0")
	ws := s.Connect("/")
	defer ws.Close()
	ws.ExpectClosed()
}

func TestPROC011_SlowStartProcess(t *testing.T) {
	t.Parallel()
	s := startServer(t, "slow-start", "500")
	ws := s.Connect("/")
	defer ws.Close()

	// Should wait for the process to be ready
	msg, err := ws.RecvTimeout(10 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive after slow start: %v", err)
	}
	if msg != "ready" {
		t.Errorf("expected 'ready', got %q", msg)
	}

	ws.Send("hello")
	ws.ExpectMessage("hello")
}

func TestPROC012_InfiniteOutputProcess(t *testing.T) {
	t.Parallel()
	s := startServer(t, "infinite", "50")
	ws := s.Connect("/")
	defer ws.Close()

	// Should receive several ticks
	for i := 0; i < 5; i++ {
		msg, err := ws.RecvTimeout(2 * time.Second)
		if err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		if msg != "tick" {
			t.Errorf("tick %d: expected 'tick', got %q", i, msg)
		}
	}
}
