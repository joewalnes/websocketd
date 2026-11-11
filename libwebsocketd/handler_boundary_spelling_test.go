// Copyright 2026 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// caseFoldingFS reports whether dir sits on a filesystem that folds case,
// which decides whether the spelling cases below can exhibit the property
// at all. On a case-sensitive filesystem "PAGE" and "page" are two genuinely
// different directories and the tests would pass vacuously, so they skip
// loudly instead.
func caseFoldingFS(t testing.TB, dir string) bool {
	t.Helper()
	probe := filepath.Join(dir, "CaseProbe")
	if err := os.MkdirAll(probe, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(probe)
	fi, err := os.Stat(filepath.Join(dir, "caseprobe"))
	return err == nil && fi.IsDir()
}

// sameFile fails the test unless a and b are the same file on disk. Every
// acceptance asserted below is paired with this, so "checkPathBoundary let
// it through" is only ever claimed about a path that really does name a
// file inside the boundary.
func sameFile(t *testing.T, a, b string) {
	t.Helper()
	fa, err := os.Stat(a)
	if err != nil {
		t.Fatalf("stat %q: %v", a, err)
	}
	fb, err := os.Stat(b)
	if err != nil {
		t.Fatalf("stat %q: %v", b, err)
	}
	if !os.SameFile(fa, fb) {
		t.Fatalf("fixture cannot exhibit the property: %q and %q are different files", a, b)
	}
}

func notSameFile(t *testing.T, a, b string) {
	t.Helper()
	fa, err := os.Stat(a)
	if err != nil {
		t.Fatalf("stat %q: %v", a, err)
	}
	fb, err := os.Stat(b)
	if err != nil {
		t.Fatalf("stat %q: %v", b, err)
	}
	if os.SameFile(fa, fb) {
		t.Fatalf("fixture is wrong: %q and %q are the same file, so the refusal below proves nothing", a, b)
	}
}

// spellingTree lays out one directory reachable under two spellings that
// differ only in case, plus a genuinely-outside directory to escape to.
//
//	<base>/PAGE/sub/x.txt            the real file
//	<base>/PAGE/alias-samecase.txt -> sub/x.txt                (control)
//	<base>/PAGE/alias-abs.txt      -> <base>/page/sub/x.txt    (absolute target)
//	<base>/PAGE/alias-rel.txt      -> ../page/sub/x.txt        (relative target)
//	<base>/OUT/loot.txt              genuinely outside
//	<base>/PAGE/escape-abs.txt     -> <base>/out/loot.txt
//	<base>/PAGE/escape-rel.txt     -> ../out/loot.txt
//
// The two symlink target forms are both built deliberately: an absolute and
// a relative target take different routes through filepath.EvalSymlinks, and
// only building both proves the answer does not depend on which the operator
// happened to write.
func spellingTree(t *testing.T) (base, page string) {
	t.Helper()
	base = t.TempDir()
	page = filepath.Join(base, "PAGE")
	if err := os.MkdirAll(filepath.Join(page, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "OUT"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(page, "sub", "x.txt"), []byte("REAL-CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "OUT", "loot.txt"), []byte("TOP SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	links := []struct{ target, name string }{
		{filepath.Join("sub", "x.txt"), "alias-samecase.txt"},
		{filepath.Join(base, "page", "sub", "x.txt"), "alias-abs.txt"},
		{filepath.Join("..", "page", "sub", "x.txt"), "alias-rel.txt"},
		{filepath.Join(base, "OUT", "loot.txt"), "escape-abs.txt"},
		{filepath.Join("..", "OUT", "loot.txt"), "escape-rel.txt"},
	}
	for _, l := range links {
		if err := os.Symlink(l.target, filepath.Join(page, l.name)); err != nil {
			t.Fatal(err)
		}
	}
	return base, page
}

// TestCheckPathBoundarySpellingOfTheSameDirectory: a file genuinely inside
// --staticdir must be served even when the route to it spells a parent
// directory differently from the way the flag spells it.
//
// checkPathBoundary ended in strings.HasPrefix over two EvalSymlinks
// results. EvalSymlinks resolves symlinks but does not canonicalize case, so
// a link whose target says "<base>/page/..." resolved to a real path that
// names a file plainly inside "<base>/PAGE" yet shares no string prefix with
// it. The check failed closed — safe, but the operator's own file 404ed.
func TestCheckPathBoundarySpellingOfTheSameDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}
	base, page := spellingTree(t)
	if !caseFoldingFS(t, base) {
		t.Skip("SKIPPING, NOT PASSING: filesystem is case-sensitive, so <base>/page and <base>/PAGE are genuinely different directories and this case cannot arise here")
	}
	real := filepath.Join(page, "sub", "x.txt")

	// Control: the same-case alias must already be accepted. If this fails,
	// the boundary is refusing everything and the assertions below would be
	// meaningless.
	sameFile(t, filepath.Join(page, "alias-samecase.txt"), real)
	if err := checkPathBoundary(filepath.Join(page, "alias-samecase.txt"), page); err != nil {
		t.Fatalf("control: same-case alias inside the boundary was refused: %v", err)
	}

	for _, name := range []string{"alias-abs.txt", "alias-rel.txt"} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(page, name)
			sameFile(t, p, real)
			if err := checkPathBoundary(p, page); err != nil {
				t.Errorf("checkPathBoundary(%q, %q) = %v, want accepted: it is the same file as %q", p, page, err, real)
			}
		})
	}

	// The boundary itself spelled the other way names the same directory,
	// so a canonically-spelled file under it is inside it.
	t.Run("boundary spelled the other way", func(t *testing.T) {
		lower := filepath.Join(base, "page")
		sameFile(t, lower, page)
		if err := checkPathBoundary(real, lower); err != nil {
			t.Errorf("checkPathBoundary(%q, %q) = %v, want accepted", real, lower, err)
		}
	})
}

// TestCheckPathBoundarySpellingDoesNotGrantEscape is the twin that must not
// trigger. Nothing about tolerating a differently-spelled parent may make a
// file outside the boundary reachable — including when the escape's target
// is spelled in the same case-variant style as the accepted aliases.
func TestCheckPathBoundarySpellingDoesNotGrantEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}
	base, page := spellingTree(t)
	loot := filepath.Join(base, "OUT", "loot.txt")
	real := filepath.Join(page, "sub", "x.txt")
	notSameFile(t, loot, real)

	for _, name := range []string{"escape-abs.txt", "escape-rel.txt"} {
		p := filepath.Join(page, name)
		sameFile(t, p, loot) // the fixture really does reach outside
		if err := checkPathBoundary(p, page); err == nil {
			t.Errorf("SECURITY: checkPathBoundary(%q, %q) accepted a file outside the boundary", p, page)
		}
	}

	// A sibling directory whose name merely shares the boundary's text, in
	// either case spelling, is still outside it.
	sibling := filepath.Join(base, "PAGE-secret")
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sibling, "loot.txt"), []byte("TOP SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, boundary := range []string{page, filepath.Join(base, "page")} {
		if err := checkPathBoundary(filepath.Join(sibling, "loot.txt"), boundary); err == nil {
			t.Errorf("SECURITY: name-prefix sibling accepted against boundary %q", boundary)
		}
	}
}

// TestCheckPathBoundaryUnresolvableRefuses pins the direction of failure.
// checkPathBoundary is used to *refuse*, so "cannot tell" must mean refuse —
// unlike the exec-directory exclusion in boundedDir.Open, where "cannot
// tell" means outside and therefore permits.
// The two share one resolution helper; they must not share one answer.
func TestCheckPathBoundaryUnresolvableRefuses(t *testing.T) {
	dir := t.TempDir()
	if err := checkPathBoundary(filepath.Join(dir, "no-such-file"), dir); err == nil {
		t.Error("SECURITY: a path that cannot be resolved should be refused")
	}
	if err := checkPathBoundary(filepath.Join(dir, "x"), filepath.Join(dir, "no-such-dir")); err == nil {
		t.Error("SECURITY: a boundary that cannot be resolved should be refused")
	}
}

// TestDirRelationDistinguishesUnknownFromOutside is the invariant the fix
// rests on. dirRelation used to answer with a bool, collapsing "outside"
// and "cannot tell" into one value. Every caller read that as safe only
// because the two happened to coincide there: the exec exclusion and the CGI mount
// prefix both *grant* on the negative, while checkPathBoundary *refuses* on
// it. Folding a refusal into a two-state answer would silently turn an
// unresolvable path into a grant somewhere.
func TestDirRelationDistinguishesUnknownFromOutside(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "in")
	if err := os.MkdirAll(filepath.Join(inside, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "out")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}

	if rel, r := dirRelation(filepath.Join(inside, "a", "b"), inside); r != relInside || rel != "a/b" {
		t.Errorf("dirRelation(<in/a/b>, <in>) = %q, %v; want %q, relInside", rel, r, "a/b")
	}
	if rel, r := dirRelation(inside, inside); r != relInside || rel != "" {
		t.Errorf("dirRelation(<in>, <in>) = %q, %v; want \"\", relInside", rel, r)
	}
	if _, r := dirRelation(outside, inside); r != relOutside {
		t.Errorf("dirRelation(<out>, <in>) = %v, want relOutside", r)
	}
	if _, r := dirRelation(filepath.Join(base, "nope"), inside); r != relUnknown {
		t.Errorf("dirRelation(<missing>, <in>) = %v, want relUnknown", r)
	}
	if _, r := dirRelation(inside, filepath.Join(base, "nope")); r != relUnknown {
		t.Errorf("dirRelation(<in>, <missing>) = %v, want relUnknown", r)
	}

	// The call sites that permit on the negative must keep reading unknown
	// as "not inside": they compare against relInside, never against
	// relOutside.
	if _, r := dirRelation(filepath.Join(base, "nope"), inside); r == relInside {
		t.Error("an unresolvable path must not compare equal to relInside")
	}
}

// TestCheckPathBoundaryFastPathImpliesIdentity proves the two acceptance
// routes cannot disagree in the accepting direction: anything the string
// prefix accepts, the identity walk accepts too. That is what makes the
// fast path a pure optimization rather than a second, looser policy — the
// walk only ever runs when the prefix test would have refused. It needs no
// case-folding filesystem, so it runs everywhere.
func TestCheckPathBoundaryFastPathImpliesIdentity(t *testing.T) {
	base := t.TempDir()
	mk := func(parts ...string) string {
		p := filepath.Join(append([]string{base}, parts...)...)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	root := filepath.Join(base, "root")
	files := []string{
		mk("root", "a.txt"),
		mk("root", "d", "b.txt"),
		mk("root", "d", "e", "c.txt"),
		mk("rootish", "x.txt"),
		mk("other", "y.txt"),
	}
	for _, f := range append(files, root, base) {
		realPath, err := filepath.EvalSymlinks(f)
		if err != nil {
			t.Fatal(err)
		}
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		prefixSays := realPath == realRoot || strings.HasPrefix(realPath, realRoot+string(filepath.Separator))
		_, r := dirRelation(realPath, realRoot)
		if prefixSays && r != relInside {
			t.Errorf("fast path accepts %q under %q but dirRelation says %v: the two disagree in the accepting direction", realPath, realRoot, r)
		}
	}
}

// BenchmarkCheckPathBoundary measures the two routes separately, because
// this predicate runs on every static and CGI request and its result cannot
// be cached — the path differs per request. The identity walk adds up to
// one os.Stat per path component, so the fast path exists to keep that off
// the common case: it must be reached without a single extra syscall, and
// the walk must run only where the old code would already have refused.
func BenchmarkCheckPathBoundary(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("symlinks require admin on Windows")
	}
	base := b.TempDir()
	page := filepath.Join(base, "PAGE")
	if err := os.MkdirAll(filepath.Join(page, "a", "b", "c"), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "OUT"), 0755); err != nil {
		b.Fatal(err)
	}
	deep := filepath.Join(page, "a", "b", "c", "x.txt")
	if err := os.WriteFile(deep, []byte("x"), 0644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "OUT", "loot.txt"), []byte("x"), 0644); err != nil {
		b.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "page", "a", "b", "c", "x.txt"), filepath.Join(page, "alias.txt")); err != nil {
		b.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "OUT", "loot.txt"), filepath.Join(page, "escape.txt")); err != nil {
		b.Fatal(err)
	}

	// The route each case takes, so a number cannot be misread: "prefix"
	// returns from the fast path, "walk" falls through to the identity walk.
	b.Run("prefix/direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := checkPathBoundary(deep, page); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("walk/case-variant", func(b *testing.B) {
		if !caseFoldingFS(b, base) {
			b.Skip("filesystem is case-sensitive; this route is unreachable here")
		}
		p := filepath.Join(page, "alias.txt")
		if err := checkPathBoundary(p, page); err != nil {
			b.Fatalf("setup: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := checkPathBoundary(p, page); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("walk/escape-refused", func(b *testing.B) {
		p := filepath.Join(page, "escape.txt")
		if err := checkPathBoundary(p, page); err == nil {
			b.Fatal("setup: escape was accepted")
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := checkPathBoundary(p, page); err == nil {
				b.Fatal("escape accepted")
			}
		}
	})
}
