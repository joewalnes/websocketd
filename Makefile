# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Self contained Go build file that will download and install (locally) the correct
# version of Go, and build our programs. Go does not need to be installed on the
# system (and if it already is, it will be ignored).

# To manually invoke the locally installed Go, use ./go

# Go installation config.
#GO_VERSION=1.2.1.linux-amd64
GO_VER=1.4
SYSTEM_NAME:=$(shell uname -s | tr '[:upper:]' '[:lower:]')
SYSTEM_ARCH:=$(shell uname -m)
GO_ARCH:=$(if $(filter x86_64, $(SYSTEM_ARCH)),amd64,386)
GO_VERSION:=$(GO_VER).$(SYSTEM_NAME)-$(GO_ARCH)$(if $(filter darwin,$(SYSTEM_NAME)),-osx10.8)
GO_DOWNLOAD_URL=http://golang.org/dl/go$(GO_VERSION).tar.gz

# Build websocketd binary
websocketd: go $(wildcard *.go) $(wildcard libwebsocketd/*.go) go-workspace/src/github.com/joewalnes/websocketd
	./go get ./go-workspace/src/github.com/joewalnes/websocketd
	./go fmt github.com/joewalnes/websocketd/libwebsocketd github.com/joewalnes/websocketd
	./go build

# Create local go workspace and symlink websocketd into the right location.
go-workspace/src/github.com/joewalnes/websocketd:
	mkdir -p go-workspace/src/github.com/joewalnes
	ln -s ../../../../ go-workspace/src/github.com/joewalnes/websocketd

# Setup ./go wrapper to use local GOPATH/GOROOT.
# Need to set PATH for gofmt.
go: go-v$(GO_VERSION)/.done
	@echo '#!/bin/sh' > $@
	@echo export PATH=$(abspath go-v$(GO_VERSION)/bin):$(PATH) >> $@
	@echo mkdir -p $(abspath go-workspace) >> $@
	@echo GOPATH=$(abspath go-workspace) GOROOT=$(abspath go-v$(GO_VERSION)) $(abspath go-v$(GO_VERSION)/bin/go) \$$@ >> $@
	chmod +x $@
	@echo 'Created ./$@ wrapper'

# Download and unpack Go distribution.
go-v$(GO_VERSION)/.done:
	mkdir -p $(dir $@)
	rm -f $@
	@echo Downloading and unpacking Go $(GO_VERSION) to $(dir $@)
	wget -q -O - $(GO_DOWNLOAD_URL) | tar xzf - --strip-components=1 -C $(dir $@)
	touch $@

# Clean up binary
clean:
	rm -rf websocketd go-workspace
.PHONY: clean

# Also clean up downloaded Go
clobber: clean
	rm -rf go $(wildcard go-v*)
.PHONY: clobber
