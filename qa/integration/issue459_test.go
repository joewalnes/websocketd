package integration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Tests for #459: forward STDERR to WebSocket clients via --passstderr.

func TestIssue459_PassStderrTagsBothStreams(t *testing.T) {
	t.Parallel()
	s := startServerOpts(t, []string{"--passstderr"}, "stderr")
	ws := s.Connect("/")
	defer ws.Close()

	collected := map[string]string{}
	deadline := time.Now().Add(5 * time.Second)
	for len(collected) < 2 && time.Now().Before(deadline) {
		msg, err := ws.RecvTimeout(1 * time.Second)
		if err != nil {
			continue
		}
		var envelope struct {
			Stream string `json:"stream"`
			Data   string `json:"data"`
		}
		if err := json.Unmarshal([]byte(msg), &envelope); err != nil {
			t.Fatalf("expected tagged JSON message, got %q: %v", msg, err)
		}
		collected[envelope.Stream] = envelope.Data
	}

	if collected["stdout"] != "stdout line" {
		t.Errorf("stdout = %q, want %q", collected["stdout"], "stdout line")
	}
	if collected["stderr"] != "stderr line" {
		t.Errorf("stderr = %q, want %q", collected["stderr"], "stderr line")
	}

	// The child's stderr must still reach websocketd's own log (on stdout),
	// same as without --passstderr - the flag adds client forwarding, it
	// doesn't replace server-side logging.
	logDeadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(s.Stdout(), "stderr line") && time.Now().Before(logDeadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(s.Stdout(), "stderr line") {
		t.Error("expected child stderr to still be logged server-side with --passstderr")
	}
}

func TestIssue459_NoPassStderrByDefault(t *testing.T) {
	// Without --passstderr, behavior is unchanged: only stdout reaches the
	// client, untagged (not wrapped in the {"stream":...} envelope).
	t.Parallel()
	s := startServer(t, "stderr")
	ws := s.Connect("/")
	defer ws.Close()

	ws.ExpectMessage("stdout line")
	ws.ExpectClosed()
}

func TestIssue459_BinaryAndPassStderrRejected(t *testing.T) {
	t.Parallel()
	_, stderr, exitCode := runWebsocketd(t, "--port=0", "--binary", "--passstderr", testcmdBin, "echo")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit combining --binary and --passstderr")
	}
	if !strings.Contains(stderr, "--binary") || !strings.Contains(stderr, "--passstderr") {
		t.Errorf("expected error mentioning both flags, got stderr: %q", stderr)
	}
}
