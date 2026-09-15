// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFile is a small helper that creates a file (and any parent
// directories) under a test's temp tree.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestBoundedDirDotSegment covers boundedDir.Open directly: every request
// whose path contains a segment beginning with "." must be refused,
// regardless of whether that segment is the whole file name, a leaf
// extension, or an intermediate directory — while ordinary names that merely
// contain a dot (a leaf extension, or a directory name with a dot in it)
// must still be served. This is the choke point every static request goes
// through (boundedDir.Open), not a single handler, so a check added here
// covers the class rather than one call site.
func TestBoundedDirDotSegment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "secret=1234")
	writeFile(t, filepath.Join(root, ".git", "config"), "gitconfig contents")
	writeFile(t, filepath.Join(root, "ok.txt"), "ordinary file")
	writeFile(t, filepath.Join(root, "file.env"), "file with dot")
	writeFile(t, filepath.Join(root, "a.b", "c.txt"), "dotted dir file")
	writeFile(t, filepath.Join(root, "sub", ".hidden", "x.txt"), "nested dotdir file")

	d := boundedDir{root: root, fs: http.Dir(root)}

	tests := []struct {
		name    string
		urlPath string
		blocked bool
	}{
		{"dotfile at root", "/.env", true},
		{"dotdir at root", "/.git/config", true},
		{"nested dotdir", "/sub/.hidden/x.txt", true},
		{"ordinary file", "/ok.txt", false},
		{"leaf with dot extension", "/file.env", false},
		{"dir name containing a dot", "/a.b/c.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := d.Open(tt.urlPath)
			if tt.blocked {
				if err == nil {
					f.Close()
					t.Fatalf("SECURITY: boundedDir.Open(%q) succeeded, want refusal", tt.urlPath)
				}
				if !os.IsNotExist(err) {
					t.Errorf("boundedDir.Open(%q) = error %v, want an is-not-exist error", tt.urlPath, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("boundedDir.Open(%q) = error %v, want success", tt.urlPath, err)
			}
			f.Close()
		})
	}
}

// TestBoundedDirDirectoryListing covers the second half of issue #476: a
// directory with no index.html must not be openable for listing purposes,
// while a directory that does have one must still work normally.
func TestBoundedDirDirectoryListing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "nolisting", "secret.txt"), "should not be listed")
	writeFile(t, filepath.Join(root, "withindex", "index.html"), "index page")
	writeFile(t, filepath.Join(root, "withindex", "other.txt"), "other file")

	d := boundedDir{root: root, fs: http.Dir(root)}

	if f, err := d.Open("/nolisting/"); err == nil {
		f.Close()
		t.Fatal("SECURITY: boundedDir.Open(\"/nolisting/\") succeeded, want refusal (no index.html)")
	} else if !os.IsNotExist(err) {
		t.Errorf(`boundedDir.Open("/nolisting/") = error %v, want an is-not-exist error`, err)
	}

	f, err := d.Open("/withindex/")
	if err != nil {
		t.Fatalf(`boundedDir.Open("/withindex/") = error %v, want success (has index.html)`, err)
	}
	f.Close()

	// A file inside the directory lacking an index.html must still be
	// reachable directly by name — only the auto-listing is blocked.
	f, err = d.Open("/nolisting/secret.txt")
	if err != nil {
		t.Fatalf(`boundedDir.Open("/nolisting/secret.txt") = error %v, want success`, err)
	}
	f.Close()
}

// TestServeStaticHTTP drives serveStatic end to end through net/http, since
// the class of bug in #476 (dotfile disclosure, auto-listing) is only fully
// observable in the actual HTTP response: status code and body, not just
// whether http.File.Open returns an error. stdout/stderr are not involved
// here; the assertions are entirely on the HTTP response.
func TestServeStaticHTTP(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "secret=1234")
	writeFile(t, filepath.Join(root, "ok.txt"), "ordinary file")
	writeFile(t, filepath.Join(root, "file.env"), "file with dot")
	writeFile(t, filepath.Join(root, "a.b", "c.txt"), "dotted dir file")
	writeFile(t, filepath.Join(root, "nolisting", "secret.txt"), "should not be listed")
	writeFile(t, filepath.Join(root, "withindex", "index.html"), "index page")

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := &WebsocketdServer{Config: &Config{StaticDir: root}, Log: log}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.serveStatic(w, req, log)
	}))
	defer srv.Close()

	get := func(p string) *http.Response {
		t.Helper()
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		return resp
	}

	if resp := get("/.env"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("SECURITY: GET /.env = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	if resp := get("/nolisting/"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("SECURITY: GET /nolisting/ = %d, want %d", resp.StatusCode, http.StatusNotFound)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), "secret.txt") {
			t.Errorf("GET /nolisting/ body leaked directory contents: %s", body)
		}
	}

	if resp := get("/ok.txt"); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /ok.txt = %d, want %d", resp.StatusCode, http.StatusOK)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "ordinary file" {
			t.Errorf("GET /ok.txt body = %q, want %q", body, "ordinary file")
		}
	}

	if resp := get("/file.env"); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /file.env = %d, want %d (dot in extension, not a segment start)", resp.StatusCode, http.StatusOK)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "file with dot" {
			t.Errorf("GET /file.env body = %q, want %q", body, "file with dot")
		}
	}

	if resp := get("/a.b/c.txt"); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /a.b/c.txt = %d, want %d (dot in dir name, not a leading dot)", resp.StatusCode, http.StatusOK)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "dotted dir file" {
			t.Errorf("GET /a.b/c.txt body = %q, want %q", body, "dotted dir file")
		}
	}

	if resp := get("/withindex/"); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /withindex/ = %d, want %d", resp.StatusCode, http.StatusOK)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "index page" {
			t.Errorf("GET /withindex/ body = %q, want %q", body, "index page")
		}
	}
}

// TestWellKnownIsExemptFromDotSegmentRule pins a deliberate decision: RFC
// 8615 reserves /.well-known/ as the standard location for URIs meant to
// be served publicly (ACME's /.well-known/acme-challenge/<token>,
// security.txt, and friends), so it is the one dot-directory this project
// exempts from the otherwise-blanket dotfile rule. The exemption is an
// exact match on the first path segment only: it does not become a
// prefix or pattern, a dot segment nested inside .well-known is still
// refused, and the symlink-boundary check still applies within it. This
// test exists so a future reader sees the exemption was decided, not
// overlooked — see the fix commit body and DIARY.md for the reasoning.
func TestWellKnownIsExemptFromDotSegmentRule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".well-known", "acme-challenge", "tok123"), "challenge response")
	writeFile(t, filepath.Join(root, ".well-known", "security.txt"), "Contact: mailto:security@example.com")
	writeFile(t, filepath.Join(root, ".well-known", ".git", "config"), "gitconfig contents")
	writeFile(t, filepath.Join(root, ".well-known-evil", "x.txt"), "not the real thing")

	outside := filepath.Join(t.TempDir(), "escaped.txt")
	writeFile(t, outside, "should never be served")
	if runtime.GOOS != "windows" {
		if err := os.Symlink(outside, filepath.Join(root, ".well-known", "escape.txt")); err != nil {
			t.Fatal(err)
		}
	}

	d := boundedDir{root: root, fs: http.Dir(root)}

	// The exempted case: an exact ACME-shaped request must be served.
	f, err := d.Open("/.well-known/acme-challenge/tok123")
	if err != nil {
		t.Fatalf(`boundedDir.Open("/.well-known/acme-challenge/tok123") = error %v, want success`, err)
	}
	f.Close()

	f, err = d.Open("/.well-known/security.txt")
	if err != nil {
		t.Fatalf(`boundedDir.Open("/.well-known/security.txt") = error %v, want success`, err)
	}
	f.Close()

	// A dot segment nested inside .well-known is still refused: the
	// exemption covers exactly the first segment, nothing deeper.
	if f, err := d.Open("/.well-known/.git/config"); err == nil {
		f.Close()
		t.Fatal(`SECURITY: boundedDir.Open("/.well-known/.git/config") succeeded, want refusal`)
	} else if !os.IsNotExist(err) {
		t.Errorf(`boundedDir.Open("/.well-known/.git/config") = error %v, want an is-not-exist error`, err)
	}

	// Not a prefix match: a directory that merely starts with
	// ".well-known" is an ordinary dotfile and stays blocked.
	if f, err := d.Open("/.well-known-evil/x.txt"); err == nil {
		f.Close()
		t.Fatal(`SECURITY: boundedDir.Open("/.well-known-evil/x.txt") succeeded, want refusal`)
	} else if !os.IsNotExist(err) {
		t.Errorf(`boundedDir.Open("/.well-known-evil/x.txt") = error %v, want an is-not-exist error`, err)
	}

	if runtime.GOOS != "windows" {
		// The symlink-boundary check is independent of the dotfile rule
		// and still applies inside the exempted directory.
		if f, err := d.Open("/.well-known/escape.txt"); err == nil {
			f.Close()
			t.Fatal(`SECURITY: boundedDir.Open("/.well-known/escape.txt") succeeded, want refusal (symlink escapes root)`)
		} else if !os.IsNotExist(err) {
			t.Errorf(`boundedDir.Open("/.well-known/escape.txt") = error %v, want an is-not-exist error`, err)
		}
	}
}

// TestServeStaticHTTPWellKnown drives the same exemption through the real
// HTTP path, since that is the shape an ACME client or the RFC 8615
// convention actually depends on.
func TestServeStaticHTTPWellKnown(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".well-known", "acme-challenge", "tok123"), "challenge response")

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := &WebsocketdServer{Config: &Config{StaticDir: root}, Log: log}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.serveStatic(w, req, log)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/.well-known/acme-challenge/tok123")
	if err != nil {
		t.Fatalf("GET /.well-known/acme-challenge/tok123: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /.well-known/acme-challenge/tok123 = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "challenge response" {
		t.Errorf("GET /.well-known/acme-challenge/tok123 body = %q, want %q", body, "challenge response")
	}
}

// TestServeStaticDirectoryNamedIndexHTML closes a gap between #476's gate
// and net/http's. boundedDir.Open lets a directory through if
// "<dir>/index.html" *opens*; http.FileServer falls back to generating a
// listing unless that name opens AND is not itself a directory. The two
// predicates disagree for exactly one input — a directory literally named
// index.html — and in the direction that produces the listing #476 exists
// to refuse, of a directory the request never named.
//
// The positive control (an ordinary directory whose index.html is a file)
// runs against the same server, so a regression that broke directory
// indexes entirely could not pass this by 404ing everything.
func TestServeStaticDirectoryNamedIndexHTML(t *testing.T) {
	root := t.TempDir()
	// x/index.html is a DIRECTORY. Its own index.html is what makes
	// boundedDir.Open accept the outer directory.
	writeFile(t, filepath.Join(root, "x", "index.html", "index.html"), "inner page")
	writeFile(t, filepath.Join(root, "x", "index.html", "LISTED-SECRET.txt"), "must not be listed")
	writeFile(t, filepath.Join(root, "ok", "index.html"), "ordinary index page")

	log := RootLogScope(LogNone, func(*LogScope, LogLevel, string, string, string, ...interface{}) {})
	h := &WebsocketdServer{Config: &Config{StaticDir: root}, Log: log}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.serveStatic(w, req, log)
	}))
	defer srv.Close()

	get := func(p string) (int, string) {
		t.Helper()
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}

	// Control: an ordinary directory index still works.
	if code, body := get("/ok/"); code != http.StatusOK || !strings.Contains(body, "ordinary index page") {
		t.Fatalf("control GET /ok/ = %d %q, want 200 with the index page", code, body)
	}

	code, body := get("/x/")
	if strings.Contains(body, "LISTED-SECRET.txt") {
		t.Errorf("SECURITY: GET /x/ = %d listed a directory: %q", code, body)
	}
	if code == http.StatusOK {
		t.Errorf("GET /x/ = 200 %q, want a refusal", body)
	}
}
