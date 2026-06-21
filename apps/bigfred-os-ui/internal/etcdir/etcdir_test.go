package etcdir

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListReadWrite(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "supervisord")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "redis.conf"), []byte("port 6380\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "supervisord.conf"), []byte("[supervisord]\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	entries, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: %+v", entries)
	}

	body, err := Read(root, "redis.conf")
	if err != nil {
		t.Fatal(err)
	}
	if body.Content != "port 6380\n" {
		t.Fatalf("content: %q", body.Content)
	}

	updated, err := Write(root, "redis.conf", "port 6380\nappendonly yes\n")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "port 6380\nappendonly yes\n" {
		t.Fatalf("updated: %q", updated.Content)
	}

	_, err = Read(root, "../passwd")
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected invalid path, got %v", err)
	}

	_, err = Read(root, "missing.conf")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
