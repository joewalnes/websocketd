// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"testing"
)

// TestTLSConfigMinVersion pins the minimum TLS version so a future refactor
// or Go default change cannot silently allow TLS 1.0/1.1.
func TestTLSConfigMinVersion(t *testing.T) {
	if got := tlsConfig().MinVersion; got != tls.VersionTLS12 {
		t.Errorf("tlsConfig().MinVersion = 0x%04x, want TLS 1.2 (0x%04x)", got, tls.VersionTLS12)
	}
}
