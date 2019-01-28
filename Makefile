# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Self contained Go build file that will download and install (locally) the correct
# version of Go, and build our programs. Go does not need to be installed on the
# system (and if it already is, it will be ignored).

# To manually invoke the locally installed Go, use ./go

# Go installation config.
GO_VER=1.11.5
SYSTEM_NAME:=$(shell uname -s | tr '[:upper:]' '[:lower:]')
SYSTEM_ARCH:=$(shell uname -m)
GO_ARCH:=$(if $(filter x86_64, $(SYSTEM_ARCH)),amd64,386)
GO_VERSION:=$(GO_VER).$(SYSTEM_NAME)-$(GO_ARCH)
GO_DOWNLOAD_URL:=http://golang.org/dl/go$(GO_VERSION).tar.gz
GO_WORKDIR:=go-v$(GO_VERSION)

# Build websocketd binary
websocketd: $(GO_WORKDIR)/bin/go $(wildcard *.go) $(wildcard libwebsocketd/*.go)
	$(GO_WORKDIR)/bin/go build

# Download and unpack Go distribution.
$(GO_WORKDIR)/bin/go:
	mkdir -p $(GO_WORKDIR)
	rm -f $@
	@echo Downloading and unpacking Go $(GO_VERSION) to $(GO_WORKDIR)
	wget -q -O - $(GO_DOWNLOAD_URL) | tar xzf - --strip-components=1 -C $(GO_WORKDIR)

# Clean up binary
clean:
	rm -rf websocketd

.PHONY: clean

# Also clean up downloaded Go
clobber: clean
	rm -rf $(wildcard go-v*)

.PHONY: clobber
