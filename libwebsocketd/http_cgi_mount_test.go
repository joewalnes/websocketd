// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// rawGet sends requestTarget verbatim on a fresh connection to baseURL's
// host. Go's http.Client (like curl) rewrites dot segments before sending,
// which would hide exactly the normalization these tests are about.
func rawGet(baseURL, requestTarget string) (*http.Response, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("tcp", u.Host, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}
	req := "GET " + requestTarget + " HTTP/1.1\r\nHost: " + u.Host + "\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return nil, err
	}
	// Drain the body before the deferred Close discards what is still in
	// flight; otherwise a "body does not contain X" assertion is vacuous for
	// any response larger than one read.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// The marker every fixture script carries in its source. If it ever comes
// back over HTTP verbatim, the static handler disclosed the script instead
// of the CGI handler executing it — issue #453.
const cgiSourceMarker = "SOURCE-MUST-NOT-BE-SERVED"

// writeMountCGIScript writes an executable /bin/sh CGI script whose *source*
// carries cgiSourceMarker and whose *output* carries body, so a test can tell
// "executed" from "disclosed" by looking at the response alone.
func writeMountCGIScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n# " + cgiSourceMarker + "\n" +
		"printf 'Content-Type: text/plain\\r\\n\\r\\n'\n" +
		"printf '" + body + "\\n'\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

// TestCgiMountPrefix covers the derivation on its own: the URL prefix at
// which --cgidir appears inside --staticdir. Every positive case is paired
// with the shape that must NOT produce a prefix, because a prefix derived
// where none exists would silently divert static URLs into the CGI handler.
func TestCgiMountPrefix(t *testing.T) {
	base := t.TempDir()
	mk := func(rel string) string {
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	page := mk("PAGE")
	cgi := mk("PAGE/cgi-bin")
	deep := mk("PAGE/a/b")
	htm := mk("PAGE/htm")
	sibling := mk("other/cgi-bin")
	// A directory whose name merely starts with the CGI directory's name.
	mk("PAGE/cgi-bindings")

	tests := []struct {
		name      string
		staticDir string
		cgiDir    string
		want      string
	}{
		{"cgi dir directly inside static dir", page, cgi, "/cgi-bin"},
		{"cgi dir nested deeper", page, deep, "/a/b"},
		{"no static dir at all", "", cgi, ""},
		{"no cgi dir at all", page, "", ""},
		{"same directory", page, page, ""},
		{"static dir inside cgi dir", htm, page, ""},
		{"disjoint siblings", htm, sibling, ""},
		{"disjoint trees", page, sibling, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cgiMountPrefix(tt.staticDir, tt.cgiDir); got != tt.want {
				t.Errorf("cgiMountPrefix(%q, %q) = %q, want %q", tt.staticDir, tt.cgiDir, got, tt.want)
			}
		})
	}
}

// TestExecExclusionMixedRelativeAndAbsolute pins the case that decides
// whether the static handler's CGI exclusion can fail open: --staticdir
// given as a relative path (websocketd --staticdir=. --cgidir=/abs/PAGE/cgi-bin
// is a perfectly ordinary command line) makes the paths the static handler
// builds relative, while --cgidir stays absolute. If dirRelation compared
// those two spellings without normalizing them first, a file plainly inside
// the CGI directory would be reported as outside it — and reported as
// outside is the permissive answer at this call site.
//
// This test manipulates the process working directory, so it must not run in
// parallel; no test in this package does.
func TestExecExclusionMixedRelativeAndAbsolute(t *testing.T) {
	base := t.TempDir()
	cgiDir := filepath.Join(base, "PAGE", "cgi-bin")
	writeFile(t, filepath.Join(cgiDir, "hello.sh"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(base, "PAGE", "index.html"), "index")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(base, "PAGE")); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	// The expression every call site uses: a definite "inside", never a
	// comparison against relOutside.
	inside := func(p, dir string) bool {
		_, r := dirRelation(p, dir)
		return r == relInside
	}

	// Relative path, absolute directory: inside.
	if !inside(filepath.Join("cgi-bin", "hello.sh"), cgiDir) {
		t.Errorf("inside(%q, %q) = false, want true", filepath.Join("cgi-bin", "hello.sh"), cgiDir)
	}
	// Absolute path, relative directory: also inside.
	if !inside(filepath.Join(cgiDir, "hello.sh"), "cgi-bin") {
		t.Errorf("inside(%q, %q) = false, want true", filepath.Join(cgiDir, "hello.sh"), "cgi-bin")
	}
	// The must-not-trigger side: a file that really is outside stays outside
	// under the same mixed spellings.
	if inside("index.html", cgiDir) {
		t.Errorf("inside(%q, %q) = true, want false", "index.html", cgiDir)
	}
	if inside(filepath.Join(base, "PAGE", "index.html"), "cgi-bin") {
		t.Errorf("inside(<abs index.html>, %q) = true, want false", "cgi-bin")
	}
}

// TestCgiURLCandidates pins which request paths get a second, prefix-stripped
// lookup inside the CGI directory. The look-alike cases matter more than the
// matches: "/cgi-bindings/…" and "/cgi-bin.txt" share the prefix as raw text
// but are not under it, and a candidate derived from either would divert a
// static URL into the CGI handler.
func TestCgiURLCandidates(t *testing.T) {
	base := t.TempDir()
	staticDir := filepath.Join(base, "PAGE")
	cgiDir := filepath.Join(staticDir, "cgi-bin")
	if err := os.MkdirAll(cgiDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		urlPath string
		want    []string
	}{
		{"/cgi-bin/hello.sh", []string{"/cgi-bin/hello.sh", "/hello.sh"}},
		{"/cgi-bin/sub/hello.sh", []string{"/cgi-bin/sub/hello.sh", "/sub/hello.sh"}},
		{"/htm/../cgi-bin/hello.sh", []string{"/cgi-bin/hello.sh", "/hello.sh"}},
		{"/cgi-bin", []string{"/cgi-bin", "/"}},
		{"/cgi-bin/", []string{"/cgi-bin", "/"}},
		// Not under the prefix, however much text they share with it.
		{"/cgi-bindings/note.txt", []string{"/cgi-bindings/note.txt"}},
		{"/cgi-bin.txt", []string{"/cgi-bin.txt"}},
		{"/hello.sh", []string{"/hello.sh"}},
		// Climbs out of the prefix, so it is not a request under it at all.
		{"/cgi-bin/../../outside/evil.sh", []string{"/outside/evil.sh"}},
	}

	for _, tt := range tests {
		t.Run(tt.urlPath, func(t *testing.T) {
			got := cgiURLCandidates(cgiMountPrefix(staticDir, cgiDir), tt.urlPath)
			if len(got) != len(tt.want) {
				t.Fatalf("cgiURLCandidates(%q) = %q, want %q", tt.urlPath, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("cgiURLCandidates(%q) = %q, want %q", tt.urlPath, got, tt.want)
				}
			}
		})
	}

	// With no static directory there is no prefix to derive, so the request
	// path is the only candidate — the pre-#453 behavior, unchanged.
	if got := cgiURLCandidates(cgiMountPrefix("", cgiDir), "/cgi-bin/hello.sh"); len(got) != 1 || got[0] != "/cgi-bin/hello.sh" {
		t.Errorf("cgiURLCandidates with no static dir = %q, want just the request path", got)
	}
}

// mountFixture builds the layout from issue #453 — a CGI directory sitting
// inside the static root — and returns a live server plus a GET helper.
//
//	PAGE/index.html
//	PAGE/cgi-bin/hello.sh        (executable)
//	PAGE/cgi-bin/.env            (a dotfile inside the CGI tree)
//	PAGE/cgi-bindings/note.txt   (name shares the CGI dir's prefix)
//	PAGE/mirror -> PAGE/cgi-bin  (symlink into the CGI tree)
//	PAGE/pub    -> PAGE/htm      (symlink that stays in the static tree)
func mountFixture(t *testing.T) (staticDir, cgiDir string, get func(string) (int, string)) {
	t.Helper()
	root := t.TempDir()
	staticDir = filepath.Join(root, "PAGE")
	cgiDir = filepath.Join(staticDir, "cgi-bin")
	writeMountCGIScript(t, filepath.Join(cgiDir, "hello.sh"), "hello-from-cgi")
	writeFile(t, filepath.Join(cgiDir, ".env"), "CGI_SECRET="+cgiSourceMarker)
	writeFile(t, filepath.Join(staticDir, "index.html"), "the index page")
	writeFile(t, filepath.Join(staticDir, "cgi-bindings", "note.txt"), "ordinary static file")
	writeFile(t, filepath.Join(staticDir, "htm", "page.html"), "a real static page")
	if err := os.Symlink(cgiDir, filepath.Join(staticDir, "mirror")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(staticDir, "htm"), filepath.Join(staticDir, "pub")); err != nil {
		t.Fatal(err)
	}

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := NewWebsocketdServer(&Config{StaticDir: staticDir, CgiDir: cgiDir}, log, 0)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// Go's http.Client rewrites dot segments before sending, which would hide
	// exactly the normalization this fix depends on, so requests go out as a
	// literal request target on a raw connection.
	get = func(target string) (int, string) {
		t.Helper()
		resp, err := rawGet(srv.URL, target)
		if err != nil {
			t.Fatalf("GET %s: %v", target, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(body)
	}
	return staticDir, cgiDir, get
}

// TestCGIDirInsideStaticDirExecutes is the issue #453 regression: with
// --staticdir=/PAGE --cgidir=/PAGE/cgi-bin, the URL a browser naturally
// forms for the script (/cgi-bin/hello.sh) must run it, not return its
// source. The must-not-trigger cases sit alongside: a sibling directory
// whose name starts with "cgi-bin", the plain static index, and a static
// file reached through a symlink that stays inside the static tree.
func TestCGIDirInsideStaticDirExecutes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture CGI scripts are /bin/sh")
	}
	_, _, get := mountFixture(t)

	t.Run("script under the cgi dir's own URL prefix runs", func(t *testing.T) {
		status, body := get("/cgi-bin/hello.sh")
		if status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
			t.Errorf("GET /cgi-bin/hello.sh = %d %q, want 200 with CGI output", status, body)
		}
		if strings.Contains(body, cgiSourceMarker) {
			t.Errorf("SECURITY: GET /cgi-bin/hello.sh disclosed script source: %q", body)
		}
	})

	t.Run("the documented direct mapping still runs", func(t *testing.T) {
		status, body := get("/hello.sh")
		if status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
			t.Errorf("GET /hello.sh = %d %q, want 200 with CGI output", status, body)
		}
	})

	t.Run("prefix is matched after normalization", func(t *testing.T) {
		status, body := get("/htm/../cgi-bin/hello.sh")
		if status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
			t.Errorf("GET /htm/../cgi-bin/hello.sh = %d %q, want 200 with CGI output", status, body)
		}
		if strings.Contains(body, cgiSourceMarker) {
			t.Errorf("SECURITY: dot-segment spelling of the CGI URL disclosed source: %q", body)
		}
	})

	t.Run("static index is untouched", func(t *testing.T) {
		// "/" rather than "/index.html": http.FileServer redirects the
		// explicit index file to the directory, and rawGet does not follow
		// redirects.
		status, body := get("/")
		if status != http.StatusOK || !strings.Contains(body, "the index page") {
			t.Errorf("GET / = %d %q, want the static index page", status, body)
		}
	})

	t.Run("a directory sharing the prefix is not diverted", func(t *testing.T) {
		status, body := get("/cgi-bindings/note.txt")
		if status != http.StatusOK || !strings.Contains(body, "ordinary static file") {
			t.Errorf("GET /cgi-bindings/note.txt = %d %q, want the static file", status, body)
		}
	})

	t.Run("a symlink that stays inside the static tree still serves", func(t *testing.T) {
		status, body := get("/pub/page.html")
		if status != http.StatusOK || !strings.Contains(body, "a real static page") {
			t.Errorf("GET /pub/page.html = %d %q, want the static file", status, body)
		}
	})

	t.Run("the cgi directory itself is not listed or served", func(t *testing.T) {
		for _, target := range []string{"/cgi-bin", "/cgi-bin/"} {
			status, body := get(target)
			if status == http.StatusOK {
				t.Errorf("GET %s = 200 %q, want a refusal", target, body)
			}
			if strings.Contains(body, "hello.sh") {
				t.Errorf("SECURITY: GET %s listed the CGI directory: %q", target, body)
			}
		}
	})
}

// TestStaticNeverDisclosesCGISource pins the security half of #453
// independently of the routing half: whatever the CGI handler declines, the
// static handler must not hand back as source. The symlink case is the one
// the prefix rule cannot reach — /mirror/hello.sh is not under /cgi-bin, so
// only a check on where the file really lives can stop it.
func TestStaticNeverDisclosesCGISource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture CGI scripts are /bin/sh")
	}
	_, _, get := mountFixture(t)

	t.Run("symlink from the static tree into the cgi dir", func(t *testing.T) {
		status, body := get("/mirror/hello.sh")
		if strings.Contains(body, cgiSourceMarker) {
			t.Errorf("SECURITY: GET /mirror/hello.sh disclosed CGI source (status %d): %q", status, body)
		}
		if status == http.StatusOK {
			t.Errorf("GET /mirror/hello.sh = 200, want a refusal")
		}
	})

	t.Run("dotfile inside the cgi dir", func(t *testing.T) {
		for _, target := range []string{"/cgi-bin/.env", "/mirror/.env"} {
			status, body := get(target)
			if strings.Contains(body, cgiSourceMarker) {
				t.Errorf("SECURITY: GET %s disclosed a file inside the CGI dir (status %d): %q", target, status, body)
			}
			if status == http.StatusOK {
				t.Errorf("GET %s = 200, want a refusal", target)
			}
		}
	})
}

// TestStaticDirInsideCGIDirStillServes is the inverse nesting
// (--cgidir=/PAGE --staticdir=/PAGE/htm). Every static file is then inside
// the CGI directory, so a blanket "never serve anything under --cgidir"
// rule would leave the static handler with nothing to serve at all. That
// configuration keeps its existing behavior.
func TestStaticDirInsideCGIDirStillServes(t *testing.T) {
	root := t.TempDir()
	cgiDir := filepath.Join(root, "PAGE")
	staticDir := filepath.Join(cgiDir, "htm")
	writeFile(t, filepath.Join(staticDir, "page.html"), "nested static page")

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := NewWebsocketdServer(&Config{StaticDir: staticDir, CgiDir: cgiDir}, log, 0)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/page.html")
	if err != nil {
		t.Fatalf("GET /page.html: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "nested static page") {
		t.Errorf("GET /page.html = %d %q, want the static page", resp.StatusCode, body)
	}
}

// TestCGIMountPrefixCannotEscape drives traversal attempts spelled through
// the new prefix. Normalization happens before the prefix is matched, so a
// path that climbs out of /cgi-bin no longer matches the prefix at all, and
// the direct mapping that then handles it still confines the result to the
// CGI directory.
func TestCGIMountPrefixCannotEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture CGI scripts are /bin/sh")
	}
	root := t.TempDir()
	staticDir := filepath.Join(root, "PAGE")
	cgiDir := filepath.Join(staticDir, "cgi-bin")
	writeMountCGIScript(t, filepath.Join(cgiDir, "ok.sh"), "ok")
	writeMountCGIScript(t, filepath.Join(root, "outside", "evil.sh"), "PWNED")
	writeFile(t, filepath.Join(root, "outside", "secret.txt"), "PWNED-"+cgiSourceMarker)

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := NewWebsocketdServer(&Config{StaticDir: staticDir, CgiDir: cgiDir}, log, 0)
	srv := httptest.NewServer(h)
	defer srv.Close()

	targets := []string{
		"/cgi-bin/../../outside/evil.sh",
		"/cgi-bin/../../outside/secret.txt",
		"/cgi-bin/%2e%2e/%2e%2e/outside/evil.sh",
		"/cgi-bin/sub/../../../outside/evil.sh",
		"/cgi-bin/..%2F..%2Foutside%2Fevil.sh",
	}
	for _, target := range targets {
		resp, err := rawGet(srv.URL, target)
		if err != nil {
			t.Fatalf("GET %s: %v", target, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), "PWNED") {
			t.Errorf("SECURITY: %q escaped the CGI directory (status %d): %q", target, resp.StatusCode, body)
		}
	}

	// Control: the legitimate script under the prefix still runs, so the
	// loop above is not passing because the whole prefix is broken.
	resp, err := rawGet(srv.URL, "/cgi-bin/ok.sh")
	if err != nil {
		t.Fatalf("GET /cgi-bin/ok.sh: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "ok") {
		t.Fatalf("control GET /cgi-bin/ok.sh = %d %q, want 200 with CGI output", resp.StatusCode, body)
	}
}

// caseInsensitiveFS reports whether dir lives on a filesystem that folds
// case, which decides whether the case-spelling cases below can exhibit the
// property at all. On a case-sensitive filesystem /PAGE and /page are two
// different directories and there is nothing to reconcile.
func caseInsensitiveFS(t *testing.T, dir string) bool {
	t.Helper()
	probe := filepath.Join(dir, "CaseProbe")
	if err := os.MkdirAll(probe, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(probe)
	fi, err := os.Stat(filepath.Join(dir, "caseprobe"))
	return err == nil && fi.IsDir()
}

// TestCgiMountPrefixSpelling covers the pairs where --staticdir and --cgidir
// name the same tree but are *spelled* differently: one through a symlink,
// one by its real path, or the two differing only in case on a
// case-insensitive filesystem. A lexical comparison of the two spellings
// reports "not nested", the mount prefix comes out empty, and the script's
// natural URL 404s — a legitimate configuration served closed.
//
// Every positive case is paired with a spelling that must still produce no
// prefix, because a prefix invented where the CGI directory is not really
// inside the static root would divert static URLs into the CGI handler.
func TestCgiMountPrefixSpelling(t *testing.T) {
	base := t.TempDir()
	mk := func(rel string) string {
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	link := func(target, name string) string {
		p := filepath.Join(base, name)
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	page := mk("PAGE")
	cgi := mk("PAGE/cgi-bin")
	elsewhere := mk("elsewhere/cgi")
	mk("PAGE/htm")

	// A release-style symlink: "current" points at the versioned directory.
	current := link(page, "current")
	// A symlink whose target is relative rather than absolute — the two
	// resolve identically but are spelled differently on disk, and only one
	// of the two forms was exercised by the pre-existing tests.
	relCurrent := filepath.Join(base, "rel-current")
	if err := os.Symlink("PAGE", relCurrent); err != nil {
		t.Fatal(err)
	}
	// A symlink sitting *inside* the static tree that points at a CGI
	// directory which really lives outside it.
	if err := os.Symlink(elsewhere, filepath.Join(page, "scripts")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		staticDir string
		cgiDir    string
		want      string
	}{
		{"static dir via symlink, cgi dir by real path", current, cgi, "/cgi-bin"},
		{"static dir by real path, cgi dir via symlink", page, filepath.Join(current, "cgi-bin"), "/cgi-bin"},
		{"both via the same symlink", current, filepath.Join(current, "cgi-bin"), "/cgi-bin"},
		{"symlink with a relative target", relCurrent, cgi, "/cgi-bin"},

		// Must-not-trigger: the same directory reached two ways is still
		// the same directory, and equal directories yield no prefix.
		{"same dir, one spelling symlinked", current, page, ""},
		{"same dir, symlinked both ways", current, relCurrent, ""},
		// Must-not-trigger: resolving spellings must not make a disjoint
		// tree look nested.
		{"cgi dir outside, reachable only via a symlink in the static tree", page, elsewhere, ""},
		{"cgi dir outside, static dir symlinked", current, elsewhere, ""},
		// Must-not-trigger: static dir inside cgi dir keeps yielding none.
		{"static dir inside cgi dir, via symlink", filepath.Join(current, "htm"), page, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cgiMountPrefix(tt.staticDir, tt.cgiDir); got != tt.want {
				t.Errorf("cgiMountPrefix(%q, %q) = %q, want %q", tt.staticDir, tt.cgiDir, got, tt.want)
			}
		})
	}

	if !caseInsensitiveFS(t, base) {
		t.Log("filesystem is case-sensitive; skipping the case-folding cases")
		return
	}
	upper := filepath.Join(base, "page")
	if got := cgiMountPrefix(page, filepath.Join(upper, "cgi-bin")); got != "/cgi-bin" {
		t.Errorf("cgiMountPrefix(%q, %q) = %q, want %q", page, filepath.Join(upper, "cgi-bin"), got, "/cgi-bin")
	}
	if got := cgiMountPrefix(upper, cgi); got != "/cgi-bin" {
		t.Errorf("cgiMountPrefix(%q, %q) = %q, want %q", upper, cgi, got, "/cgi-bin")
	}
	// Must-not-trigger: a case variant of the *same* directory is still the
	// same directory, so still no prefix.
	if got := cgiMountPrefix(page, upper); got != "" {
		t.Errorf("cgiMountPrefix(%q, %q) = %q, want %q", page, upper, got, "")
	}
}

// TestCgiMountPrefixUnresolvable pins the fail-closed direction: a directory
// that cannot be resolved yields no prefix, so routing falls back to the
// long-standing direct mapping rather than inventing one. websocketd's own
// startup rejects a --cgidir or --staticdir that does not exist, so this is
// reachable only through direct library use.
func TestCgiMountPrefixUnresolvable(t *testing.T) {
	base := t.TempDir()
	page := filepath.Join(base, "PAGE")
	if err := os.MkdirAll(filepath.Join(page, "cgi-bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := cgiMountPrefix(page, filepath.Join(page, "nope")); got != "" {
		t.Errorf("cgiMountPrefix with a missing cgi dir = %q, want %q", got, "")
	}
	if got := cgiMountPrefix(filepath.Join(base, "nope"), filepath.Join(page, "cgi-bin")); got != "" {
		t.Errorf("cgiMountPrefix with a missing static dir = %q, want %q", got, "")
	}
	// A file, not a directory, on either side.
	f := filepath.Join(base, "afile")
	writeFile(t, f, "x")
	if got := cgiMountPrefix(f, filepath.Join(page, "cgi-bin")); got != "" {
		t.Errorf("cgiMountPrefix with a file as static dir = %q, want %q", got, "")
	}
}

// symlinkedMountFixture is mountFixture's layout with one difference that is
// the whole point: --staticdir is handed to the server through a symlink
// while --cgidir is given by its real path, the shape a release symlink
// produces. Routing must be identical to the unsymlinked case.
func symlinkedMountFixture(t *testing.T) (get func(string) (int, string)) {
	t.Helper()
	root := t.TempDir()
	real := filepath.Join(root, "releases", "v1")
	cgiDir := filepath.Join(real, "cgi-bin")
	writeMountCGIScript(t, filepath.Join(cgiDir, "hello.sh"), "hello-from-cgi")
	writeFile(t, filepath.Join(real, "index.html"), "the index page")
	writeFile(t, filepath.Join(real, "cgi-bindings", "note.txt"), "ordinary static file")
	writeMountCGIScript(t, filepath.Join(root, "outside", "evil.sh"), "PWNED")
	writeFile(t, filepath.Join(root, "outside", "secret.txt"), "PWNED-"+cgiSourceMarker)

	current := filepath.Join(root, "current")
	if err := os.Symlink(real, current); err != nil {
		t.Fatal(err)
	}

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := NewWebsocketdServer(&Config{StaticDir: current, CgiDir: cgiDir}, log, 0)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return func(target string) (int, string) {
		t.Helper()
		resp, err := rawGet(srv.URL, target)
		if err != nil {
			t.Fatalf("GET %s: %v", target, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(body)
	}
}

// TestSymlinkedStaticDirRoutesCGI is the end-to-end half: with --staticdir
// given as a release symlink and --cgidir as the real directory inside it,
// /cgi-bin/hello.sh must run the script. Before the identity-based
// derivation it returned 404 — the static handler correctly refused to
// disclose an exec directory's contents, and the CGI handler never
// recognized the URL, so a working configuration served nothing.
func TestSymlinkedStaticDirRoutesCGI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture CGI scripts are /bin/sh")
	}
	get := symlinkedMountFixture(t)

	t.Run("script under the cgi dir's URL prefix runs", func(t *testing.T) {
		status, body := get("/cgi-bin/hello.sh")
		if strings.Contains(body, cgiSourceMarker) {
			t.Errorf("SECURITY: GET /cgi-bin/hello.sh disclosed script source (status %d): %q", status, body)
		}
		if status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
			t.Errorf("GET /cgi-bin/hello.sh = %d %q, want 200 with CGI output", status, body)
		}
	})

	// Positive controls: without these, a server that had simply stopped
	// serving anything would pass every refusal assertion below.
	t.Run("control: the direct mapping runs", func(t *testing.T) {
		status, body := get("/hello.sh")
		if status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
			t.Errorf("GET /hello.sh = %d %q, want 200 with CGI output", status, body)
		}
	})
	t.Run("control: static content is served through the symlinked root", func(t *testing.T) {
		status, body := get("/")
		if status != http.StatusOK || !strings.Contains(body, "the index page") {
			t.Errorf("GET / = %d %q, want the static index page", status, body)
		}
	})

	t.Run("a sibling sharing the prefix text is not diverted", func(t *testing.T) {
		status, body := get("/cgi-bindings/note.txt")
		if status != http.StatusOK || !strings.Contains(body, "ordinary static file") {
			t.Errorf("GET /cgi-bindings/note.txt = %d %q, want the static file", status, body)
		}
	})

	t.Run("the cgi directory is neither listed nor served", func(t *testing.T) {
		for _, target := range []string{"/cgi-bin", "/cgi-bin/"} {
			status, body := get(target)
			if status == http.StatusOK {
				t.Errorf("GET %s = 200 %q, want a refusal", target, body)
			}
			if strings.Contains(body, "hello.sh") {
				t.Errorf("SECURITY: GET %s listed the CGI directory: %q", target, body)
			}
		}
	})

	t.Run("traversal through the derived prefix cannot escape", func(t *testing.T) {
		targets := []string{
			"/cgi-bin/../../outside/evil.sh",
			"/cgi-bin/../../outside/secret.txt",
			"/cgi-bin/%2e%2e/%2e%2e/outside/evil.sh",
			"/cgi-bin/..%2F..%2Foutside%2Fevil.sh",
			"/cgi-bin/sub/../../../outside/evil.sh",
		}
		for _, target := range targets {
			status, body := get(target)
			if strings.Contains(body, "PWNED") {
				t.Errorf("SECURITY: %q escaped the tree (status %d): %q", target, status, body)
			}
		}
	})
}

// TestCGIDirOutsideStaticTreeStaysRefused is the adversarial twin of the fix.
// Resolving spellings must not turn "the CGI directory is reachable from the
// static tree through a symlink" into "the CGI directory is inside the static
// tree". --cgidir here really lives outside --staticdir; the only thing
// joining them is a symlink named "scripts". /scripts/hello.sh must stay
// refused — both as CGI (no prefix is derived) and as static content (the
// file lives in an exec directory, and the symlink leaves the static root).
// The script's documented URL, /hello.sh, keeps working.
func TestCGIDirOutsideStaticTreeStaysRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture CGI scripts are /bin/sh")
	}
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	cgiDir := filepath.Join(root, "elsewhere", "cgi")
	writeMountCGIScript(t, filepath.Join(cgiDir, "hello.sh"), "hello-from-cgi")
	writeFile(t, filepath.Join(staticDir, "index.html"), "the index page")
	if err := os.Symlink(cgiDir, filepath.Join(staticDir, "scripts")); err != nil {
		t.Fatal(err)
	}

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := NewWebsocketdServer(&Config{StaticDir: staticDir, CgiDir: cgiDir}, log, 0)
	srv := httptest.NewServer(h)
	defer srv.Close()
	get := func(target string) (int, string) {
		t.Helper()
		resp, err := rawGet(srv.URL, target)
		if err != nil {
			t.Fatalf("GET %s: %v", target, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(body)
	}

	// Controls first: prove the server really is serving from both roots,
	// so the refusals below are refusals and not a dead server.
	if status, body := get("/hello.sh"); status != http.StatusOK || !strings.Contains(body, "hello-from-cgi") {
		t.Fatalf("control GET /hello.sh = %d %q, want 200 with CGI output", status, body)
	}
	if status, body := get("/"); status != http.StatusOK || !strings.Contains(body, "the index page") {
		t.Fatalf("control GET / = %d %q, want the static index page", status, body)
	}

	if status, body := get("/scripts/hello.sh"); status == http.StatusOK {
		t.Errorf("GET /scripts/hello.sh = 200 %q, want a refusal", body)
	} else if strings.Contains(body, cgiSourceMarker) {
		t.Errorf("SECURITY: GET /scripts/hello.sh disclosed CGI source (status %d): %q", status, body)
	}
}
