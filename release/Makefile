# Copyright 2013-2019 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Uses Semantic Versioning scheme - http://semver.org/
VERSION_MAJOR=0
VERSION_MINOR=3

# Last part of version number (patch) is incremented automatically from Git tags
LAST_PATCH_VERSION:=$(shell git ls-remote git@github.com:joewalnes/websocketd.git \
		| grep -e 'refs/tags/v[0-9\.]*$$' \
		| sed -e 's|^.*refs/tags/v||' \
		| grep -e "^$(VERSION_MAJOR)\.$(VERSION_MINOR)\.[0-9][0-9]*$$" \
		| sed -e 's/^.*\.//' \
		| sort -n \
		| tail -n 1)



VERSION_PATCH:=$(shell expr $$(( $(or $(LAST_PATCH_VERSION),-1) + 1 )))
RELEASE_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_PATCH)

GO_VERSION=1.11.5
PLATFORMS=linux_amd64 linux_386 linux_arm linux_arm64 darwin_amd64 freebsd_amd64 freebsd_386 windows_386 windows_amd64 openbsd_386 openbsd_amd64 solaris_amd64


# Would NOT WORK on: ARM, WINDOWS

UNAME_SYS=$(shell uname -s | tr A-Z a-z)
UNAME_ARCH=$(shell uname -m)
ifeq ($(UNAME_ARCH),x86_64)
	UNAME_ARCH=amd64
else
	UNAME_ARCH=386 
endif

ifeq ($(UNAME_SYS),gnu)
	UNAME_SYS=linux
else ifeq ($(UNAME_SYS),sunos)
	UNAME_SYS=solaris
endif



GO_DOWNLOAD_URL=https://dl.google.com/go/go$(GO_VERSION).$(UNAME_SYS)-$(UNAME_ARCH).tar.gz
GO_DIR=../go-$(GO_VERSION)

# Prevent any global environment polluting the builds
FLAGS_linux_amd64   = GOOS=linux   GOARCH=amd64
FLAGS_linux_386     = GOOS=linux   GOARCH=386
FLAGS_linux_arm     = GOOS=linux   GOARCH=arm   GOARM=5 # ARM5 support for Raspberry Pi
FLAGS_linux_arm64   = GOOS=linux   GOARCH=arm64  # no need for GOARM= (which is technically 8)
FLAGS_darwin_amd64  = GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0
FLAGS_darwin_386    = GOOS=darwin  GOARCH=386   CGO_ENABLED=0
FLAGS_freebsd_amd64 = GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0
FLAGS_freebsd_386   = GOOS=freebsd GOARCH=386   CGO_ENABLED=0
FLAGS_windows_386   = GOOS=windows GOARCH=386   CGO_ENABLED=0
FLAGS_windows_amd64 = GOOS=windows GOARCH=amd64 CGO_ENABLED=0
FLAGS_openbsd_386   = GOOS=openbsd GOARCH=386   CGO_ENABLED=0
FLAGS_openbsd_amd64 = GOOS=openbsd GOARCH=amd64 CGO_ENABLED=0
FLAGS_solaris_amd64 = GOOS=solaris GOARCH=amd64 CGO_ENABLED=0

all: build

localgo: $(GO_DIR)/bin/go

$(GO_DIR)/bin/go:
	mkdir -p $(GO_DIR)
	rm -f $@
	@echo Downloading and unpacking Go $(GO_VERSION) to $(GO_DIR)
	curl -s $(GO_DOWNLOAD_URL) | tar xzf - --strip-components=1 -C $(GO_DIR)


# Cross-compile final applications
out/$(RELEASE_VERSION)/%/websocketd: ../*.go ../libwebsocketd/*.go $(GO_DIR)/bin/go
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/bin/go build -ldflags "-X main.version=$(RELEASE_VERSION)" -o out/$(RELEASE_VERSION)/$*/websocketd ..

out/$(RELEASE_VERSION)/%/websocketd.exe: ../*.go ../libwebsocketd/*.go $(GO_DIR)/bin/go
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/bin/go build -ldflags "-X main.version=$(RELEASE_VERSION)" -o out/$(RELEASE_VERSION)/$*/websocketd.exe ..

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)-%.zip: out/$(RELEASE_VERSION)/%/websocketd
	rm -f $@
	zip -j $@ out/$(RELEASE_VERSION)/$*/* ../{README.md,LICENSE,CHANGES}


BINARIES = $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/websocketd$(EXTENSION_$(PLATFORM)))
ZIPS = $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)-$(PLATFORM).zip)
DEBS = out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_i386.deb out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_amd64.deb
RPMS = out/$(RELEASE_VERSION)/websocketd.$(RELEASE_VERSION).i386.rpm out/$(RELEASE_VERSION)/websocketd.$(RELEASE_VERSION).x86_64.rpm

binaries: $(BINARIES)

build: out/$(RELEASE_VERSION)/CHECKSUMS

out/$(RELEASE_VERSION)/CHECKSUMS: $(BINARIES) $(ZIPS)  $(DEBS) $(RPMS)
	sha256sum $^ | sed -e 's/out\/$(RELEASE_VERSION)\///' >$@



BASEFPM=--description "WebSockets server that converts STDIO scripts to powerful HTML5 applications." --url http://websocketd.com/ --license MIT  --vendor "websocketd team <joe@walnes.com>" --maintainer "abc@alexsergeyev.com"

DEBFPM=""
RPMFPM=--rpm-os linux

deb: $(DEBS)

rpm: $(RPMS)


out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_i386.deb: $(GO_UNPACKED) out/$(RELEASE_VERSION)/linux_386/websocketd
	mkdir -p out/$(RELEASE_VERSION)/deb32/{usr/bin,usr/share/man/man1,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_386/websocketd out/$(RELEASE_VERSION)/deb32/usr/bin/
	cp ../{LICENSE,AUTHORS,CHANGES,README.md} out/$(RELEASE_VERSION)/deb32/usr/share/doc/websocketd-$(RELEASE_VERSION)
	cat websocketd.man | gzip > out/$(RELEASE_VERSION)/deb32/usr/share/man/man1/websocket.1.gz
	fpm -f -s dir -t deb -a i386 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/deb32/ -p out/$(RELEASE_VERSION)/websocketd-VERSION_ARCH.deb $(BASEFPM) $(DEB_FPM) usr/
	rm -rf out/$(RELEASE_VERSION)/deb32/

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_amd64.deb:  $(GO_UNPACKED) out/$(RELEASE_VERSION)/linux_amd64/websocketd
	mkdir -p out/$(RELEASE_VERSION)/deb64/{usr/bin,usr/share/man/man1,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_amd64/websocketd out/$(RELEASE_VERSION)/deb64/usr/bin/
	cp ../{LICENSE,AUTHORS,CHANGES,README.md} out/$(RELEASE_VERSION)/deb64/usr/share/doc/websocketd-$(RELEASE_VERSION)
	cat websocketd.man | gzip > out/$(RELEASE_VERSION)/deb64/usr/share/man/man1/websocket.1.gz
	fpm -f -s dir -t deb -a amd64 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/deb64/ -p out/$(RELEASE_VERSION)/websocketd-VERSION_ARCH.deb $(BASEFPM) $(DEB_FPM) usr/
	rm -rf out/$(RELEASE_VERSION)/deb64/

out/$(RELEASE_VERSION)/websocketd.$(RELEASE_VERSION).x86_64.rpm: $(GO_UNPACKED) out/$(RELEASE_VERSION)/linux_amd64/websocketd
	mkdir -p out/$(RELEASE_VERSION)/rpm64/{usr/bin,etc/default,usr/share/man/man1,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_amd64/websocketd out/$(RELEASE_VERSION)/rpm64/usr/bin/
	cp ../{LICENSE,AUTHORS,CHANGES,README.md} out/$(RELEASE_VERSION)/rpm64/usr/share/doc/websocketd-$(RELEASE_VERSION)
	cat websocketd.man | gzip > out/$(RELEASE_VERSION)/rpm64/usr/share/man/man1/websocket.1.gz
	fpm -f -s dir -t rpm -a x86_64 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/rpm64/ -p out/$(RELEASE_VERSION)/websocketd.VERSION.ARCH.rpm $(BASEFPM) $(RPMFPM) usr/
	rm -rf out/$(RELEASE_VERSION)/rpm64/

out/$(RELEASE_VERSION)/websocketd.$(RELEASE_VERSION).i386.rpm: $(GO_UNPACKED) out/$(RELEASE_VERSION)/linux_386/websocketd
	mkdir -p out/$(RELEASE_VERSION)/rpm32/{usr/bin,etc/default,usr/share/man/man1,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_386/websocketd out/$(RELEASE_VERSION)/rpm32/usr/bin/
	cp ../{LICENSE,AUTHORS,CHANGES,README.md} out/$(RELEASE_VERSION)/rpm32/usr/share/doc/websocketd-$(RELEASE_VERSION)
	cat websocketd.man | gzip > out/$(RELEASE_VERSION)/rpm32/usr/share/man/man1/websocket.1.gz
	fpm -f -s dir -t rpm -a i386 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/rpm32/ -p out/$(RELEASE_VERSION)/websocketd.VERSION.ARCH.rpm $(BASEFPM) $(RPMFPM) usr/
	rm -rf out/$(RELEASE_VERSION)/rpm32/


# Clean up
clobber: clean
	rm -rf $(GO_DIR)


clean:
	rm -rf out

.PHONY: all build deb rpm localgo clobber clean
