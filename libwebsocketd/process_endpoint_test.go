// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"
)

func quietLogScope() *LogScope {
	return RootLogScope(LogFatal, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
}

// echoProcess launches a short-lived process that prints one line and exits.
func echoProcess(t *testing.T) *LaunchedProcess {
	t.Helper()
	var name string
	var args []string
	if runtime.GOOS == "windows" {
		name, args = "cmd.exe", []string{"/c", "echo hello"}
	} else {
		name, args = "/bin/echo", []string{"hello"}
	}
	lp, err := launchCmd(name, args, nil)
	if err != nil {
		t.Fatalf("launchCmd failed: %v", err)
	}
	return lp
}

// stdoutStderrProcess launches a short-lived process that writes one line to
// each of STDOUT and STDERR, then exits.
func stdoutStderrProcess(t *testing.T, stdoutLine, stderrLine string) *LaunchedProcess {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}
	lp, err := launchCmd("/bin/sh", []string{"-c", "echo " + stdoutLine + "; echo " + stderrLine + " >&2"}, nil)
	if err != nil {
		t.Fatalf("launchCmd failed: %v", err)
	}
	return lp
}

// TestStderrLongLineKeepsProcessAlive reproduces a remote-triggered hang: a
// stderr write larger than the reader's 4KB buffer with no trailing newline
// made ReadSlice return ErrBufferFull, which both stderr pumps treated as
// fatal. The pump quit, the child then blocked forever on its next stderr
// write once the OS pipe filled, and the session was wedged.
//
// The child here writes 256KB (more than any common pipe buffer) of
// un-newlined stderr, then a stdout marker, then serves stdin. If the pump
// died, the child never reaches the marker.
func TestStderrLongLineKeepsProcessAlive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}
	script := `head -c 262144 /dev/zero | tr '\0' 'X' >&2; echo MARKER; while IFS= read -r line; do echo "R:$line"; done`
	lp, err := launchCmd("/bin/sh", []string{"-c", script}, nil)
	if err != nil {
		t.Fatalf("launchCmd failed: %v", err)
	}

	pe := NewProcessEndpoint(lp, false, false, quietLogScope(), false)
	pe.StartReading()
	defer pe.Terminate()

	// With the bug the child wedges writing stderr and this times out.
	if msg, ok := recvTimeout(t, pe.Output(), 10*time.Second); !ok || string(msg) != "MARKER" {
		t.Fatalf("stdout MARKER never arrived (child wedged on stderr write?): got %q, ok=%v", msg, ok)
	}

	// The session must stay fully functional afterwards.
	if !pe.Send([]byte("ping\n")) {
		t.Fatal("Send to stdin failed")
	}
	if msg, ok := recvTimeout(t, pe.Output(), 10*time.Second); !ok || string(msg) != "R:ping" {
		t.Fatalf("stdin roundtrip failed after long stderr line: got %q, ok=%v", msg, ok)
	}
}

// TestStderrLongLineKeepsProcessAlive_PassStderr is the --passstderr mirror:
// the tagged stderr reader must keep draining (delivering the long line as
// consecutive chunks) instead of abandoning the pipe.
func TestStderrLongLineKeepsProcessAlive_PassStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}
	script := `head -c 262144 /dev/zero | tr '\0' 'X' >&2; echo MARKER; while IFS= read -r line; do echo "R:$line"; done`
	lp, err := launchCmd("/bin/sh", []string{"-c", script}, nil)
	if err != nil {
		t.Fatalf("launchCmd failed: %v", err)
	}

	pe := NewProcessEndpoint(lp, false, false, quietLogScope(), true)
	pe.StartReading()
	defer pe.Terminate()

	stderrBytes := 0
	deadline := time.After(15 * time.Second)
	for {
		select {
		case data, ok := <-pe.Output():
			if !ok {
				t.Fatal("output channel closed before MARKER arrived")
			}
			var envelope taggedMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("failed to parse JSON message %q: %v", data, err)
			}
			if envelope.Stream == "stdout" {
				if envelope.Data == "MARKER" {
					goto alive
				}
				t.Fatalf("unexpected stdout message %q", envelope.Data)
			}
			stderrBytes += len(envelope.Data)
		case <-deadline:
			t.Fatalf("MARKER never arrived (child wedged on stderr write?); %d stderr bytes relayed", stderrBytes)
		}
	}

alive:
	// The session must stay fully functional afterwards.
	if !pe.Send([]byte("ping\n")) {
		t.Fatal("Send to stdin failed")
	}
	for {
		select {
		case data, ok := <-pe.Output():
			if !ok {
				t.Fatal("output channel closed before R:ping arrived")
			}
			var envelope taggedMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("failed to parse JSON message %q: %v", data, err)
			}
			if envelope.Stream == "stdout" && envelope.Data == "R:ping" {
				goto complete
			}
		case <-time.After(10 * time.Second):
			t.Fatal("stdin roundtrip failed after long stderr line")
		}
	}

complete:
	// MARKER races ahead of the last stderr chunks on the shared output
	// channel, so completeness is checked here, after the roundtrip: the
	// full 256KB written before MARKER must have been relayed as chunks.
	for stderrBytes < 250000 {
		select {
		case data, ok := <-pe.Output():
			if !ok {
				t.Fatalf("output channel closed with only %d of 262144 stderr bytes relayed", stderrBytes)
			}
			var envelope taggedMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("failed to parse JSON message %q: %v", data, err)
			}
			if envelope.Stream == "stderr" {
				stderrBytes += len(envelope.Data)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out with only %d of 262144 stderr bytes relayed (pump quit early?)", stderrBytes)
		}
	}
}

// recvTimeout reads one message from ch, failing the test on timeout.
func recvTimeout(t *testing.T, ch chan []byte, timeout time.Duration) ([]byte, bool) {
	t.Helper()
	select {
	case msg, ok := <-ch:
		return msg, ok
	case <-time.After(timeout):
		return nil, false
	}
}

// TestTerminateUnblocksParkedReader reproduces the goroutine leak that occurs
// when the relay stops draining Output() (e.g. the WebSocket send failed) while
// the stdout reader is parked on the unbuffered output channel send. Terminate
// kills the process, which only unblocks reads, not channel sends — the reader
// must observe termination and exit on its own.
func TestTerminateUnblocksParkedReader(t *testing.T) {
	for _, mode := range []struct {
		name string
		bin  bool
	}{{"text", false}, {"binary", true}} {
		t.Run(mode.name, func(t *testing.T) {
			before := runtime.NumGoroutine()

			pe := NewProcessEndpoint(echoProcess(t), false, mode.bin, quietLogScope(), false)
			pe.StartReading()

			// Never drain pe.Output(): the reader picks up "hello" and parks
			// on the channel send, exactly like a relay whose peer went away.
			// Give it a moment to reach that state.
			time.Sleep(100 * time.Millisecond)

			pe.Terminate()

			// All endpoint goroutines (stdout reader, stderr logger, waiter)
			// must exit once the endpoint is terminated.
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if runtime.NumGoroutine() <= before {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Fatalf("goroutine leak: %d goroutines before, %d after Terminate (stdout reader parked on output send?)",
				before, runtime.NumGoroutine())
		})
	}
}

// TestTerminateUnblocksParkedReader_PassStderr is the --passstderr mirror of
// TestTerminateUnblocksParkedReader: relayStdout and relayStderr (in their
// --passstderr, tagged mode) must each observe Terminate's done signal and
// exit, the same as the plain text/binary readers.
func TestTerminateUnblocksParkedReader_PassStderr(t *testing.T) {
	before := runtime.NumGoroutine()

	pe := NewProcessEndpoint(stdoutStderrProcess(t, "out1", "err1"), false, false, quietLogScope(), true)
	pe.StartReading()

	// Never drain pe.Output(): both taggers read their one line and park on
	// the channel send, exactly like a relay whose peer went away.
	time.Sleep(100 * time.Millisecond)

	pe.Terminate()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine leak: %d goroutines before, %d after Terminate (tagged reader parked on output send?)",
		before, runtime.NumGoroutine())
}

func TestPassStderrTagging(t *testing.T) {
	pe := NewProcessEndpoint(stdoutStderrProcess(t, "stdout-msg", "stderr-msg"), false, false, quietLogScope(), true)
	pe.StartReading()
	defer pe.Terminate()

	collected := map[string]string{}
	timeout := time.After(5 * time.Second)
	for len(collected) < 2 {
		select {
		case data, ok := <-pe.Output():
			if !ok {
				t.Fatalf("output channel closed early with %d/2 messages: %v", len(collected), collected)
			}
			var envelope taggedMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("failed to parse JSON message %q: %v", data, err)
			}
			collected[envelope.Stream] = envelope.Data
		case <-timeout:
			t.Fatalf("timeout waiting for messages, got %d/2: %v", len(collected), collected)
		}
	}

	if collected["stdout"] != "stdout-msg" {
		t.Errorf("stdout = %q, want %q", collected["stdout"], "stdout-msg")
	}
	if collected["stderr"] != "stderr-msg" {
		t.Errorf("stderr = %q, want %q", collected["stderr"], "stderr-msg")
	}

	// The channel must close only after both readers are done, never while
	// one might still send.
	select {
	case _, ok := <-pe.Output():
		if ok {
			t.Fatal("expected no further messages")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for output channel to close")
	}
}

func TestPassStderrTagMessageEscaping(t *testing.T) {
	msg := tagMessage("stderr", []byte(`quote " backslash \ newline`+"\n"+"tab\ttab"))
	var envelope taggedMessage
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("tagMessage produced invalid JSON %q: %v", msg, err)
	}
	if envelope.Stream != "stderr" {
		t.Errorf("stream = %q, want %q", envelope.Stream, "stderr")
	}
	want := "quote \" backslash \\ newline\ntab\ttab"
	if envelope.Data != want {
		t.Errorf("data = %q, want %q", envelope.Data, want)
	}
}
