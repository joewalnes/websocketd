// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
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

			pe := NewProcessEndpoint(echoProcess(t), mode.bin, quietLogScope())
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
