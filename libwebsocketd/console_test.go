// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newConsoleServer starts a real HTTP server serving only the dev console.
func newConsoleServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := &Config{DevConsole: true, StartupTime: time.Now()}
	h := NewWebsocketdServer(cfg, RootLogScope(LogError, func(l *LogScope, level LogLevel, levelName string, category string, msg string, args ...interface{}) {
	}), 0)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// rawConsoleGet sends a request target and Host header verbatim, without the
// normalization Go's http.Client (or curl) would apply. Reflected-injection
// bugs only show up on the wire form.
func rawConsoleGet(t *testing.T, addr, target, host string) (*http.Response, string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	req := "GET " + target + " HTTP/1.1\r\nHost: " + host + "\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write request: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response for %q: %v", target, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read body for %q: %v", target, err)
	}
	return resp, string(body)
}

// TestDevConsoleBodyIsRequestIndependent pins the structural fix for the
// reflected-XSS class: the console page must not depend on the request at
// all. Not "is escaped" - "is not there". A body that varies with the
// request target or the Host header is, by construction, a body an attacker
// has a say in.
func TestDevConsoleBodyIsRequestIndependent(t *testing.T) {
	srv := newConsoleServer(t)
	addr := strings.TrimPrefix(srv.URL, "http://")

	cases := []struct {
		name, target, host string
	}{
		{"root", "/", addr},
		{"other path", "/anything", addr},
		{"deep path with query", "/a/b/c?x=1&y=2", addr},
		{"quote break-out in target", `/"><script>alert(1)</script>`, addr},
		// A Host header containing '"' never reaches the handler - net/http
		// answers 400 first - so the reachable hostile case is a well-formed
		// but attacker-chosen host, which is what the old page echoed.
		{"attacker-chosen host", "/", "evil.example.com:31337"},
		{"host with no port", "/", "example.com"},
	}

	want := ""
	for _, tc := range cases {
		resp, body := rawConsoleGet(t, addr, tc.target, tc.host)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: expected 200, got %d", tc.name, resp.StatusCode)
		}
		if strings.Contains(body, "alert(1)") {
			t.Errorf("SECURITY: %s: request-controlled text reached the console body", tc.name)
		}
		if want == "" {
			want = body
			continue
		}
		if body != want {
			t.Errorf("%s: console body differs from the /-response (%d bytes vs %d); the page must be a constant",
				tc.name, len(body), len(want))
		}
	}
	if want != ConsoleContent {
		t.Errorf("served body is not ConsoleContent (%d bytes served, %d bytes embedded)", len(want), len(ConsoleContent))
	}
}

// TestDevConsoleSecurityHeaders moved to console_serving_test.go, merged
// with a stronger CSP check that parses the header into directives rather
// than substring-matching it (see that file's version of this test for
// why: a substring match can pass against a malformed policy that happens
// to contain the right words in the wrong place). This file's Content-Type,
// ETag-strength and unsafe-eval assertions were folded into it there.

// TestDevConsoleETagIsHonoured checks the strong validator actually
// short-circuits a repeat fetch. An ETag nothing revalidates against is
// decoration.
func TestDevConsoleETagIsHonoured(t *testing.T) {
	srv := newConsoleServer(t)
	first, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	io.Copy(io.Discard, first.Body)
	first.Body.Close()
	etag := first.Header.Get("ETag")
	if etag == "" {
		t.Fatal("no ETag to revalidate with")
	}

	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Header.Set("If-None-Match", etag)
	second, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conditional get: %v", err)
	}
	defer second.Body.Close()
	body, _ := io.ReadAll(second.Body)
	if second.StatusCode != http.StatusNotModified {
		t.Errorf("conditional GET with If-None-Match returned %d (%d body bytes), want 304", second.StatusCode, len(body))
	}
}

// inlineBlocks returns the inner text of every <tag>...</tag> block in the
// page. The CSP hashes must cover exactly these.
func inlineBlocks(content, tag string) []string {
	open, closing := "<"+tag+">", "</"+tag+">"
	var out []string
	rest := content
	for {
		i := strings.Index(rest, open)
		if i < 0 {
			return out
		}
		rest = rest[i+len(open):]
		j := strings.Index(rest, closing)
		if j < 0 {
			return out
		}
		out = append(out, rest[:j])
		rest = rest[j+len(closing):]
	}
}

// TestConsoleCSPHashesCoverContent recomputes the hashes from the page that
// is actually served and requires the CSP to name every one. A CSP whose
// hashes were typed by hand goes stale the first time the page is edited,
// and a stale hash means a blank console.
func TestConsoleCSPHashesCoverContent(t *testing.T) {
	srv := newConsoleServer(t)
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	csp := resp.Header.Get("Content-Security-Policy")

	for _, tag := range []string{"script", "style"} {
		blocks := inlineBlocks(string(body), tag)
		if len(blocks) == 0 {
			t.Fatalf("found no inline <%s> blocks in the served page; this test cannot reach its subject", tag)
		}
		for i, b := range blocks {
			sum := sha256.Sum256([]byte(b))
			want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
			if !strings.Contains(csp, want) {
				t.Errorf("CSP does not cover inline <%s> block %d (hash %s); browsers will refuse to run it.\nCSP: %s",
					tag, i, want, csp)
			}
		}
		// Every hash in the corresponding directive must belong to a block:
		// a leftover hash is a hash that was not derived from the content.
		dir := tag + "-src "
		k := strings.Index(csp, dir)
		if k < 0 {
			t.Fatalf("CSP has no %s directive", dir)
		}
		seg := csp[k+len(dir):]
		if e := strings.Index(seg, ";"); e >= 0 {
			seg = seg[:e]
		}
		for _, tok := range strings.Fields(seg) {
			if !strings.HasPrefix(tok, "'sha256-") {
				t.Errorf("%s directive contains non-hash source %q", dir, tok)
				continue
			}
			found := false
			for _, b := range blocks {
				sum := sha256.Sum256([]byte(b))
				if tok == "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'" {
					found = true
				}
			}
			if !found {
				t.Errorf("%s hash %s matches no inline block in the page (stale hand-written hash?)", dir, tok)
			}
		}
	}
}

// TestConsoleMakesNoExternalRequests moved to console_serving_test.go as
// TestDevConsoleNoExternalResources, backed by findExternalResourceRefs - a
// position-aware parser (only flags src=/href=/url()/@import, not any
// mention of "http" in text) with its own false-positive/false-negative
// unit tests, replacing this file's fixed-substring scan.

// TestConsoleSizeBudget guards the "single file, no build step" promise. The
// number is a budget, not a law - but a console that quietly grows past it
// has stopped being the thing that was agreed.
func TestConsoleSizeBudget(t *testing.T) {
	const budget = 26 * 1024
	if n := len(consoleHTML); n > budget {
		t.Errorf("console.html is %d bytes, over the %d byte budget", n, budget)
	}
	if len(ConsoleContent) < 500 {
		t.Fatalf("console content is %d bytes; this test cannot reach its subject", len(ConsoleContent))
	}
}

// TestConsoleHasNoTemplatePlaceholders: every {{...}} must be resolved at
// init. A surviving placeholder means either a broken page or a
// substitution still happening per request.
func TestConsoleHasNoTemplatePlaceholders(t *testing.T) {
	if i := strings.Index(ConsoleContent, "{{"); i >= 0 {
		end := i + 40
		if end > len(ConsoleContent) {
			end = len(ConsoleContent)
		}
		t.Errorf("unresolved template placeholder in the served console: %q", ConsoleContent[i:end])
	}
}

// TestConsoleDOMContract pins the ids the browser tests drive. Renaming one
// silently is how a UI test suite starts passing against nothing.
func TestConsoleDOMContract(t *testing.T) {
	for _, id := range []string{
		"url", "connect", "status", "send", "sendbtn", "frames", "detail",
		"detail-opcode", "detail-size", "detail-body", "clear", "theme",
		"counters", "count-sent", "count-recv", "count-bytes", "empty",
	} {
		if !strings.Contains(ConsoleContent, `id="`+id+`"`) {
			t.Errorf("console is missing the contracted element id=%q", id)
		}
	}
	if !strings.Contains(ConsoleContent, `data-theme="auto"`) {
		t.Error(`console <html> should ship with data-theme="auto"`)
	}
}

// TestDevConsoleDisabledStaysDisabled is the control: with the flag off,
// nothing above should be reachable at all.
func TestDevConsoleDisabledStaysDisabled(t *testing.T) {
	cfg := &Config{DevConsole: false, StartupTime: time.Now()}
	h := NewWebsocketdServer(cfg, RootLogScope(LogError, func(l *LogScope, level LogLevel, levelName string, category string, msg string, args ...interface{}) {
	}), 0)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 404 {
		t.Errorf("without --devconsole, expected 404, got %d: %s", resp.StatusCode, string(body))
	}
	if resp.Header.Get("Content-Security-Policy") != "" {
		t.Error("console CSP leaked onto a non-console response")
	}
}
