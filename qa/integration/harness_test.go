// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHarnessReadyRejectsForeignListener covers the race that made
// TestENV008_UniqueID flaky on CI.
//
// freePort has to close its probe listener before websocketd can bind the
// port, so another process can take it in between. websocketd exits when it
// cannot bind, but a plain TCP dial to that port still succeeds — it connects
// to whoever holds it. Readiness therefore has to prove that *our* server is
// up, not merely that something is listening; otherwise the harness hands
// back a server that is not running and the test fails later with an
// unexplained connection reset and no captured output to explain it.
func TestHarnessReadyRejectsForeignListener(t *testing.T) {
	t.Parallel()

	// Hold the port so websocketd cannot have it.
	squatter, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer squatter.Close()
	port := squatter.Addr().(*net.TCPAddr).Port

	s := startServerRawArgs(t, []string{
		"--port=" + strconv.Itoa(port),
		"--address=127.0.0.1",
		"--loglevel=access",
		testcmdBin, "echo",
	})

	err = s.waitReady(port, 10*time.Second)
	if err == nil {
		t.Fatal("waitReady reported the server ready while a foreign listener held the port and websocketd had exited")
	}
	// The failure must explain itself: websocketd logs the bind error before
	// exiting, and that log is what turns a mystifying reset into a diagnosis.
	if !strings.Contains(err.Error(), "address already in use") &&
		!strings.Contains(err.Error(), "Only one usage of each socket address") {
		t.Errorf("waitReady error should quote websocketd's bind failure, got: %v", err)
	}
}

// TestHarnessReadyDetectsExit checks the same guard for a server that exits
// during startup for any other reason — a bad flag, a missing command. The
// harness should say so rather than poll a dead port until the timeout.
func TestHarnessReadyDetectsExit(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	s := startServerRawArgs(t, []string{
		"--port=" + strconv.Itoa(port),
		"--address=127.0.0.1",
		"--nosuchflag",
		testcmdBin, "echo",
	})

	start := time.Now()
	err := s.waitReady(port, 10*time.Second)
	if err == nil {
		t.Fatal("waitReady reported ready for a websocketd that never started")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("waitReady took %v to notice the process had exited; it should return as soon as it does", elapsed)
	}
}

// TestHarnessFreePortNoReissue checks that two servers in the same test
// binary are never handed the same port. The OS usually cycles ephemeral
// ports so this rarely collided on its own, but nothing prevented it: the
// probe listener is closed before the port is used.
func TestHarnessFreePortNoReissue(t *testing.T) {
	t.Parallel()

	seen := make(map[int]bool)
	for i := 0; i < 50; i++ {
		port := freePort(t)
		if seen[port] {
			t.Fatalf("freePort returned port %d twice", port)
		}
		seen[port] = true
	}
}
