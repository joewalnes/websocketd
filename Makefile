# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

websocketd: $(wildcard *.go) $(wildcard libwebsocketd/*.go)
	go fmt github.com/joewalnes/websocketd/libwebsocketd github.com/joewalnes/websocketd
	go build

clean:
	rm -f websocketd
.PHONY: clean
