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

			pe := NewProcessEndpoint(echoProcess(t), mode.bin, quietLogScope(), false)
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
// TestTerminateUnblocksParkedReader: readStdoutTagged and readStderrTagged
// must each observe Terminate's done signal and exit, the same as the plain
// text/binary readers.
func TestTerminateUnblocksParkedReader_PassStderr(t *testing.T) {
	before := runtime.NumGoroutine()

	pe := NewProcessEndpoint(stdoutStderrProcess(t, "out1", "err1"), false, quietLogScope(), true)
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
	pe := NewProcessEndpoint(stdoutStderrProcess(t, "stdout-msg", "stderr-msg"), false, quietLogScope(), true)
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
