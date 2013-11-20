# Copyright 2013 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Uses Semantic Versioning scheme - http://semver.org/
VERSION_MAJOR=0
VERSION_MINOR=2

# Last part of version number (patch) is incremented automatically from Git tags
LAST_PATCH_VERSION:=$(shell git ls-remote git@github.com:joewalnes/websocketd.git \
		| grep -e 'refs/tags/v[0-9\.]*$$' \
		| sed -e 's|^.*refs/tags/v||' \
		| grep -e "^$(VERSION_MAJOR)\.$(VERSION_MINOR)\.[0-9][0-9]*$$" \
		| sed -e 's/^.*\.//' \
		| sort -n \
		| tail -n 1)
VERSION_PATCH:=$(if $(LAST_PATCH_VERSION),$(shell expr $(LAST_PATCH_VERSION) + 1),0)
RELEASE_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_PATCH)

GO_VERSION=1.1.2
PLATFORMS=linux_amd64 linux_386 linux_arm darwin_amd64 windows_386 windows_amd64

GO_SRC_URL=https://go.googlecode.com/files/go$(GO_VERSION).src.tar.gz
GO_DOWNLOAD=go-local/$(GO_VERSION).tgz
GO_DIR=go-local/$(GO_VERSION)
GO_UNPACKED=$(GO_DIR)/.unpacked

# Prevent any global environment polluting the builds
GOROOT=$(shell readlink -f $(GO_DIR))/go
GOPATH=$(shell readlink -f go-path)

FLAGS_all = GOROOT=$(GOROOT) GOPATH=$(GOPATH)
FLAGS_linux_amd64   = $(FLAGS_all) GOOS=linux   GOARCH=amd64
FLAGS_linux_386     = $(FLAGS_all) GOOS=linux   GOARCH=386
FLAGS_linux_arm     = $(FLAGS_all) GOOS=linux   GOARCH=arm   GOARM=5 # ARM5 support for Raspberry Pi
FLAGS_darwin_amd64  = $(FLAGS_all) GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0
FLAGS_windows_386   = $(FLAGS_all) GOOS=windows GOARCH=386   CGO_ENABLED=0
FLAGS_windows_amd64 = $(FLAGS_all) GOOS=windows GOARCH=amd64 CGO_ENABLED=0

EXTENSION_windows_386=.exe
EXTENSION_windows_amd64=.exe


all: release
.PHONY: all

go-path/src/github.com/joewalnes/websocketd:
	mkdir -p go-path/src/github.com/joewalnes
	ln -s ../../../../../ go-path/src/github.com/joewalnes/websocketd

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
out/$(RELEASE_VERSION)/%/.built: $(GO_DIR)/go/pkg/%/.built $(wildcard ../*.go) go-path/src/github.com/joewalnes/websocketd
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/go/bin/go get ./go-path/src/github.com/joewalnes/websocketd
	$(FLAGS_$*) $(GO_DIR)/go/bin/go build -ldflags "-X main.version $(RELEASE_VERSION)" -o out/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*) $(wildcard ../*.go)
	touch $@

build: $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/.built)
.PHONY: build


# release to http://download.websocketd.com/
out/$(RELEASE_VERSION)/%/.released: out/$(RELEASE_VERSION)/%/.built
	rm -f $@
	s3cmd put --acl-public --mime-type=application/octet-stream out/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*) s3://download.websocketd.com/releases/websocketd/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*)
	touch $@

release: $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/.released) tag
.PHONY: release 


# Create remote Git tag
tag:
	git tag -a v$(RELEASE_VERSION) -m 'Tagged $(RELEASE_VERSION)'
	git push git@github.com:joewalnes/websocketd.git --tags
.PHONY: tag


# Clean up
clean:
	rm -rf out
.PHONY: clean


clobber: clean
	rm -rf go-local
.PHONY: clobber
