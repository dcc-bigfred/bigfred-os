package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_missingLogRootExitsClean(t *testing.T) {
	dir := t.TempDir()
	if err := run(config{logRoot: filepath.Join(dir, "missing")}); err != nil {
		t.Fatal(err)
	}
}

func TestRotateFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "svc.log")
	if err := os.WriteFile(logPath, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rotateFile(logPath); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 0 {
		t.Fatalf("log not truncated: size=%d", st.Size())
	}
	matches, err := filepath.Glob(filepath.Join(dir, "svc.log.*.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one .gz archive, got %v", matches)
	}
}

func TestDeleteExpiredGzip(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.gz")
	if err := os.WriteFile(old, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	cutoff := time.Now().AddDate(0, 0, -14)
	if err := deleteExpiredGzip(dir, cutoff); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expected old.gz to be deleted")
	}
}
