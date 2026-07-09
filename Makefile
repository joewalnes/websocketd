# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Builds websocketd using the system Go toolchain.
# See go.mod for the minimum required Go version.

websocketd: $(wildcard *.go) $(wildcard libwebsocketd/*.go)
	go build

clean:
	rm -rf websocketd

.PHONY: clean
