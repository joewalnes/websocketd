# Copyright 2013-2019 Joe Walnes and the websocketd team.
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

RELEASE_VERSION=0.3.2

PLATFORMS=linux_amd64 linux_386 linux_arm linux_arm64 darwin_amd64 freebsd_amd64 freebsd_386 windows_386 windows_amd64 openbsd_386 openbsd_amd64 solaris_amd64

GO_VER:=1.11.5
GO_DIR:=../go-$(GO_VER)

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


BUILDINFO:=$(shell echo "Built $(shell date +"%F %R") by $(shell git config user.email) on $(shell uname -v) git:$(shell git rev-parse --verify HEAD | cut -b 1-7)" | tr " " ^)


out/$(RELEASE_VERSION)/%/websocketd: ../*.go ../libwebsocketd/*.go $(GO_DIR)/bin/go
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/bin/go build -ldflags "-X main.version=$(RELEASE_VERSION)" -ldflags "-X main.buildinfo=$(BUILDINFO)" -o out/$(RELEASE_VERSION)/$*/websocketd ..

out/$(RELEASE_VERSION)/%/websocketd.exe: ../*.go ../libwebsocketd/*.go $(GO_DIR)/bin/go
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/bin/go build -ldflags "-X main.version=$(RELEASE_VERSION)" -ldflags "-X main.buildinfo=$(BUILDINFO)" -o out/$(RELEASE_VERSION)/$*/websocketd.exe ..

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


clean:
	rm -rf out

clobber:
	rm -rf $(GO_DIR)


.PHONY: all build deb rpm localgo clean clobber


include ../golang.mk
