# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

RELEASE_VERSION=0.0.0.0

GO_VERSION=1.0.3
PLATFORMS=linux_amd64 linux_386 linux_arm darwin_amd64 windows_386 windows_amd64

GO_SRC_URL=https://go.googlecode.com/files/go$(GO_VERSION).src.tar.gz
GO_DOWNLOAD=go-local/$(GO_VERSION).tgz
GO_DIR=go-local/$(GO_VERSION)
GO_UNPACKED=$(GO_DIR)/.unpacked

# Prevent any global environment polluting the builds
GOROOT=
GOPATH=

FLAGS_linux_amd64   = GOOS=linux   GOARCH=amd64
FLAGS_linux_386     = GOOS=linux   GOARCH=386
FLAGS_linux_arm     = GOOS=linux   GOARCH=arm   GOARM=5 # ARM5 support for Raspberry Pi
FLAGS_darwin_amd64  = GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0
FLAGS_windows_386   = GOOS=windows GOARCH=386   CGO_ENABLED=0
FLAGS_windows_amd64 = GOOS=windows GOARCH=amd64 CGO_ENABLED=0

EXTENSION_windows_386=.exe
EXTENSION_windows_amd64=.exe


all: apps
.PHONY: all


# Download Go source code
$(GO_DOWNLOAD):
	mkdir -p $(dir $@)
	curl $(GO_SRC_URL) > $@

go-download: $(GO_DOWNLOAD)
.PHONY: go-download


# Unpack Go source code
$(GO_UNPACKED): $(GO_DOWNLOAD)
	rm -f $(GO_UNPACKED)
	mkdir -p $(GO_DIR)
	tar xvzf $(GO_DOWNLOAD) -C $(GO_DIR)
	touch $(GO_UNPACKED)

go-unpack: $(GO_UNPACKED)
.PHONY: go-unpack


# Cross-compile Go package for target platform
$(GO_DIR)/go/pkg/%/.built: $(GO_UNPACKED)
	@echo "===========  Cross compiling GO for $*"
	rm -f $@
	cd $(GO_DIR)/go/src && $(FLAGS_$*) ./make.bash --no-clean
	touch $@

go-compile: $(foreach PLATFORM,$(PLATFORMS),$(GO_DIR)/go/pkg/$(PLATFORM)/.built)
.PHONY: go-compile


# Cross-compile final applications
out/$(RELEASE_VERSION)/%/.built: $(GO_DIR)/go/pkg/%/.built $(wildcard ../*.go)
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/go/bin/go get ..
	$(FLAGS_$*) $(GO_DIR)/go/bin/go build -ldflags "-X main.version $(RELEASE_VERSION)" -o out/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*) $(wildcard ../*.go)
	touch $@

apps: $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/.built)
.PHONY: apps


# Upload to http://download.websocketd.com/
out/$(RELEASE_VERSION)/%/.uploaded: out/$(RELEASE_VERSION)/%/.built
	rm -f $@
	s3cmd put --acl-public --mime-type=application/octet-stream out/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*) s3://download.websocketd.com/releases/websocketd/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*)
	touch $@

upload: $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/.uploaded)
.PHONY: upload


# Clean up
clean:
	rm -rf out
.PHONY: clean

clobber: clean
	rm -rf go-local
.PHONY: clobber
