// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

// This value can be set for releases at build time using:
//   go {build|run} -ldflags "-X main.version 1.2.3.4".
// If unset, Version() shall return "DEVBUILD".
var version string

func Version() string {
	if version == "" {
		return "DEVBUILD"
	}
	return version
}
