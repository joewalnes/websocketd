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

VERSION_PATCH:=$(if $(LAST_PATCH_VERSION),$(shell expr $(LAST_PATCH_VERSION)),0)
RELEASE_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_PATCH)

GO_VERSION=1.3.1
PLATFORMS=linux_amd64 linux_386 linux_arm darwin_amd64 darwin_386 freebsd_amd64 freebsd_386 windows_386 windows_amd64

GO_SRC_URL=https://storage.googleapis.com/golang/go$(GO_VERSION).src.tar.gz
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
FLAGS_darwin_386    = $(FLAGS_all) GOOS=darwin  GOARCH=386   CGO_ENABLED=0
FLAGS_freebsd_amd64  = $(FLAGS_all) GOOS=freebsd  GOARCH=amd64 CGO_ENABLED=0
FLAGS_freebsd_386    = $(FLAGS_all) GOOS=freebsd  GOARCH=386   CGO_ENABLED=0
FLAGS_windows_386   = $(FLAGS_all) GOOS=windows GOARCH=386   CGO_ENABLED=0
FLAGS_windows_amd64 = $(FLAGS_all) GOOS=windows GOARCH=amd64 CGO_ENABLED=0

EXTENSION_windows_386=.exe
EXTENSION_windows_amd64=.exe

msg=@/bin/echo -e "\x1b[33;01m>>> $1\x1b[0m"

all: build deb rpm
.PHONY: all

go-path/src/github.com/joewalnes/websocketd:
	mkdir -p go-path/src/github.com/joewalnes
	ln -s ../../../../../ go-path/src/github.com/joewalnes/websocketd

# Download Go source code
$(GO_DOWNLOAD):
	$(call msg,"Dowloading Go $(GO_VERSION)")
	mkdir -p $(dir $@)
	curl --silent --fail --output $@ $(GO_SRC_URL)

go-download: $(GO_DOWNLOAD)
.PHONY: go-download


# Unpack Go source code
$(GO_UNPACKED): $(GO_DOWNLOAD)
	$(call msg,"Unpacking Go $(GO_VERSION)")
	rm -f $(GO_UNPACKED)
	mkdir -p $(GO_DIR)
	tar xzf $(GO_DOWNLOAD) -C $(GO_DIR)
	touch $(GO_UNPACKED)

go-unpack: $(GO_UNPACKED)
.PHONY: go-unpack


# Cross-compile Go package for target platform
$(GO_DIR)/go/pkg/%/.built: $(GO_UNPACKED)
	$(call msg,"Building Go for $*")
	cd $(GO_DIR)/go/src && $(FLAGS_$*) ./make.bash --no-clean >/dev/null 2>&1
	touch $@

go-compile: $(foreach PLATFORM,$(PLATFORMS),$(GO_DIR)/go/pkg/$(PLATFORM)/.built)
.PHONY: go-compile


# Cross-compile final applications
out/$(RELEASE_VERSION)/%/websocketd: $(GO_DIR)/go/pkg/%/.built $(wildcard ../*.go) go-path/src/github.com/joewalnes/websocketd
	$(call msg,"Compiling release for $*")
	rm -f $@
	mkdir -p $(dir $@)
	$(FLAGS_$*) $(GO_DIR)/go/bin/go get ./go-path/src/github.com/joewalnes/websocketd
	$(FLAGS_$*) $(GO_DIR)/go/bin/go build -ldflags "-X main.version $(RELEASE_VERSION)" -o out/$(RELEASE_VERSION)/$*/websocketd$(EXTENSION_$*) $(wildcard ../*.go)
	touch $@

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)-%.zip: out/$(RELEASE_VERSION)/%/websocketd $(wildcard ../*.go) go-path/src/github.com/joewalnes/websocketd
	rm -f $@
	zip -j $@ out/$(RELEASE_VERSION)/$*/* ../{INSTALL,README.md,LICENSE}


build: $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/$(PLATFORM)/websocketd) $(foreach PLATFORM,$(PLATFORMS),out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)-$(PLATFORM).zip)
.PHONY: build


BASEFPM=--description "WebSockets server that converts STDIO scripts to powerful HTML5 applications." --url http://websocketd.com/ --license MIT  --vendor "websocketd team <joe@walnes.com>" --maintainer "abc@alexsergeyev.com"

DEBFPM=""
RPMFPM=--rpm-os linux

deb: out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_386.deb out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_amd64.deb

rpm: out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION).i386.rpm out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION).x86_64.rpm


out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_386.deb: $(GO_DIR)/go/pkg/linux_386/.built out/$(RELEASE_VERSION)/linux_386/websocketd
	mkdir -p out/$(RELEASE_VERSION)/deb32/{usr/bin,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_386/websocketd out/$(RELEASE_VERSION)/deb32/usr/bin/
	cp ../{LICENSE,AUTHORS,README.md} out/$(RELEASE_VERSION)/deb32/usr/share/doc/websocketd-$(RELEASE_VERSION)
	fpm -f -s dir -t deb -a 386 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/deb32/ -p out/$(RELEASE_VERSION)/websocketd-VERSION_ARCH.deb $(BASEFPM) $(DEB_FPM) usr/
	rm -rf out/$(RELEASE_VERSION)/deb32/

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION)_amd64.deb:  $(GO_DIR)/go/pkg/linux_amd64/.built out/$(RELEASE_VERSION)/linux_amd64/websocketd
	mkdir -p out/$(RELEASE_VERSION)/deb64/{usr/bin,usr/share/doc/websocketd-$(RELEASE_VERSION)}
	cp out/$(RELEASE_VERSION)/linux_amd64/websocketd out/$(RELEASE_VERSION)/deb64/usr/bin/
	cp ../{LICENSE,AUTHORS,README.md} out/$(RELEASE_VERSION)/deb64/usr/share/doc/websocketd-$(RELEASE_VERSION)
	fpm -f -s dir -t deb -a amd64 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/deb64/ -p out/$(RELEASE_VERSION)/websocketd-VERSION_ARCH.deb $(BASEFPM) $(DEB_FPM) usr/
	rm -rf out/$(RELEASE_VERSION)/deb64/

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION).x86_64.rpm: $(GO_DIR)/go/pkg/linux_amd64/.built out/$(RELEASE_VERSION)/linux_amd64/websocketd
	mkdir -p out/$(RELEASE_VERSION)/rpm64/{usr/bin,etc/default,usr/libexec,usr/share/websocketd,usr/share/doc/websocketd-$(RELEASE_VERSION),usr/lib/systemd/system}
	cp out/$(RELEASE_VERSION)/linux_amd64/websocketd out/$(RELEASE_VERSION)/rpm64/usr/bin/
	cp ../{LICENSE,AUTHORS,README.md} out/$(RELEASE_VERSION)/rpm64/usr/share/doc/websocketd-$(RELEASE_VERSION)
	fpm -f -s dir -t rpm -a x86_64 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/rpm64/ -p out/$(RELEASE_VERSION)/websocketd.VERSION.ARCH.rpm $(BASEFPM) $(RPMFPM) usr/
	rm -rf out/$(RELEASE_VERSION)/rpm64/

out/$(RELEASE_VERSION)/websocketd-$(RELEASE_VERSION).i386.rpm: $(GO_DIR)/go/pkg/linux_386/.built out/$(RELEASE_VERSION)/linux_386/websocketd
	mkdir -p out/$(RELEASE_VERSION)/rpm32/{usr/bin,etc/default,usr/libexec,usr/share/websocketd,usr/share/doc/websocketd-$(RELEASE_VERSION),usr/lib/systemd/system}
	cp out/$(RELEASE_VERSION)/linux_386/websocketd out/$(RELEASE_VERSION)/rpm32/usr/bin/
	cp ../{LICENSE,AUTHORS,README.md} out/$(RELEASE_VERSION)/rpm32/usr/share/doc/websocketd-$(RELEASE_VERSION)
	fpm -f -s dir -t rpm -a i386 -n websocketd -v $(RELEASE_VERSION) -C out/$(RELEASE_VERSION)/rpm32/ -p out/$(RELEASE_VERSION)/websocketd.VERSION.ARCH.rpm $(BASEFPM) $(RPMFPM) usr/
	rm -rf out/$(RELEASE_VERSION)/rpm32/




# Clean up
clean:
	rm -rf out go-path
.PHONY: clean


clobber: clean
	rm -rf go-local
.PHONY: clobber
