// Copyright 2013 Joe Walnes and the websocketd team.
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

func TestProcessEndpointStderrForwarding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}

	log := RootLogScope(LogNone, func(logScope *LogScope, level LogLevel, levelName string, category string, msg string, args ...interface{}) {})

	t.Run("stderr forwarded as JSON when passStderr enabled", func(t *testing.T) {
		lp, err := launchCmd("/bin/sh", []string{"-c", "echo stdout-msg; echo stderr-msg >&2; exit 0"}, []string{})
		if err != nil {
			t.Fatalf("launch failed: %v", err)
		}

		pe := NewProcessEndpoint(lp, false, log, true)
		pe.StartReading()

		collected := make(map[string]string)
		timeout := time.After(5 * time.Second)
		expected := 2

		for len(collected) < expected {
			select {
			case data, ok := <-pe.Output():
				if !ok {
					t.Fatalf("output channel closed early, only got %d messages (expected %d)", len(collected), expected)
				}
				var envelope struct {
					Stream string `json:"stream"`
					Data   string `json:"data"`
				}
				if err := json.Unmarshal(data, &envelope); err != nil {
					t.Fatalf("failed to parse JSON message: %v (got %q)", err, string(data))
				}
				collected[envelope.Stream] = envelope.Data

			case <-timeout:
				t.Fatalf("timeout waiting for messages, got %d (expected %d)", len(collected), expected)
			}
		}

		if collected["stdout"] != "stdout-msg" {
			t.Errorf("expected stdout 'stdout-msg', got %q", collected["stdout"])
		}
		if collected["stderr"] != "stderr-msg" {
			t.Errorf("expected stderr 'stderr-msg', got %q", collected["stderr"])
		}

		pe.Terminate()
	})

	t.Run("stderr NOT forwarded when passStderr disabled", func(t *testing.T) {
		lp, err := launchCmd("/bin/sh", []string{"-c", "echo stdout-msg; echo stderr-msg >&2; exit 0"}, []string{})
		if err != nil {
			t.Fatalf("launch failed: %v", err)
		}

		pe := NewProcessEndpoint(lp, false, log, false)
		pe.StartReading()

		timeout := time.After(5 * time.Second)
		var messages []string

		for {
			select {
			case data, ok := <-pe.Output():
				if !ok {
					pe.Terminate()

					for _, msg := range messages {
						if msg == "stderr-msg" {
							t.Error("stderr should NOT be forwarded when passStderr is disabled")
						}
					}
					if len(messages) != 1 || messages[0] != "stdout-msg" {
						t.Errorf("expected exactly ['stdout-msg'], got %v", messages)
					}
					return
				}
				messages = append(messages, string(data))

			case <-timeout:
				t.Fatalf("timeout after receiving %d messages", len(messages))
			}
		}
	})

	t.Run("output channel closes after both stdout and stderr complete", func(t *testing.T) {
		lp, err := launchCmd("/bin/sh", []string{"-c", "echo out1; echo err1 >&2; echo out2; echo err2 >&2; exit 0"}, []string{})
		if err != nil {
			t.Fatalf("launch failed: %v", err)
		}

		pe := NewProcessEndpoint(lp, false, log, true)
		pe.StartReading()

		count := 0
		timeout := time.After(5 * time.Second)

		for {
			select {
			case _, ok := <-pe.Output():
				if !ok {
					if count != 4 {
						t.Errorf("expected 4 messages total, got %d", count)
					}
					pe.Terminate()
					return
				}
				count++

			case <-timeout:
				t.Fatalf("timeout after receiving %d messages", count)
			}
		}
	})
}
