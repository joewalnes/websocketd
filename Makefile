# Copyright 2013-2019 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.


GO_VER:=1.11.5
GO_DIR:=go-$(GO_VER)

BUILDINFO:=$(shell echo "Built $(shell date +"%F %R") by $(shell git config user.email) on $(shell uname -v) git:$(shell git rev-parse --verify HEAD | cut -b 1-7)" | tr " " ^)

# Build websocketd binary
websocketd: $(GO_DIR)/bin/go $(wildcard *.go) $(wildcard libwebsocketd/*.go)
	$(GO_DIR)/bin/go build -ldflags "-X main.buildinfo=$(BUILDINFO)" .

clean: clean
	rm -rf websocketd

clobber: clean localclean
	rm -rf $(GO_DIR)

.PHONY: clean clobber


include golang.mk
