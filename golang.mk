# Copyright 2013-2019 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Go installation config.
SYSTEM_NAME:=$(shell uname -s | tr '[:upper:]' '[:lower:]')
SYSTEM_ARCH:=$(shell uname -m)
GO_ARCH:=$(if $(filter x86_64, $(SYSTEM_ARCH)),amd64,386)
GO_VERSION:=$(GO_VER).$(SYSTEM_NAME)-$(GO_ARCH)
GO_DOWNLOAD_URL:=https://dl.google.com/go/go$(GO_VERSION).tar.gz

localgo: $(GO_DIR)/bin/go

# Download and unpack Go distribution.
$(GO_DIR)/bin/go:
	mkdir -p $(GO_DIR)
	rm -f $@
	@echo Downloading and unpacking Go $(GO_VERSION) to $(GO_DIR)
	curl -s $(GO_DOWNLOAD_URL) | tar xf - --strip-components=1 -C $(GO_DIR)
	# prevent scanning of go.mod/go.sum
	touch $(GO_DIR)/go.mod 
