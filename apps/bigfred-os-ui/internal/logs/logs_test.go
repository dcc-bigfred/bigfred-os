package logs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathRejectsTraversal(t *testing.T) {
	roots := []string{t.TempDir()}
	if _, err := ResolvePath(roots, "tmp:../etc/passwd"); err == nil {
		t.Fatal("expected error for traversal")
	}
}

func TestListAndTail(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "redis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "redis.log")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ListAll([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries: %+v", entries)
	}
	if entries[0].Name != "redis.log" {
		t.Fatalf("entry: %+v", entries[0])
	}

	slug := rootSlug(root)
	pathResolved, err := ResolvePath([]string{root}, entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if pathResolved != path {
		t.Fatalf("resolved %q want %q", pathResolved, path)
	}
	_ = slug

	lines, err := TailLast(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "line2" || lines[1] != "line3" {
		t.Fatalf("tail: %#v", lines)
	}
}

func TestListVarLogStyleFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "syslog"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.gz"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ListAll([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "syslog" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestParseRootsDefaults(t *testing.T) {
	got := ParseRoots("", "")
	want := []string{"/data/logs", "/var/log"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v", got)
	}
}

func TestParseRootsCommaSeparated(t *testing.T) {
	got := ParseRoots("/data/logs,/var/log", "")
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}
