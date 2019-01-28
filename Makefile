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
GO_DOWNLOAD_URL:=https://dl.google.com/go/go$(GO_VERSION).tar.gz
GO_DIR:=go-$(GO_VER)

# Build websocketd binary
websocketd: $(GO_DIR)/bin/go $(wildcard *.go) $(wildcard libwebsocketd/*.go)
	$(GO_DIR)/bin/go build

localgo: $(GO_DIR)/bin/go

# Download and unpack Go distribution.
$(GO_DIR)/bin/go:
	mkdir -p $(GO_DIR)
	rm -f $@
	@echo Downloading and unpacking Go $(GO_VERSION) to $(GO_DIR)
	curl -s $(GO_DOWNLOAD_URL) | tar xf - --strip-components=1 -C $(GO_DIR)

# Clean up binary
clean:
	rm -rf websocketd

.PHONY: clean

# Also clean up downloaded Go
clobber: clean
	rm -rf $(wildcard go-v*)

.PHONY: clobber
