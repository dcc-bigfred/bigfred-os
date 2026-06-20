package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestListInitScripts(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, filepath.Join(dir, "S30-redis"), "#!/bin/sh\n")
	writeExec(t, filepath.Join(dir, "S35-victoriametrics"), "#!/bin/sh\n")
	if err := os.WriteFile(filepath.Join(dir, "rcS"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "S99-bad"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d services: %+v", len(list), list)
	}
	if list[0].ID != "redis" || list[1].ID != "victoriametrics" {
		t.Fatalf("order/ids: %+v", list)
	}
}

func TestControlRejectsInvalidAction(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, filepath.Join(dir, "S10-demo"), "#!/bin/sh\necho ok\n")
	if err := Control(dir, "demo", "pause"); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("expected ErrInvalidAction, got %v", err)
	}
}

func TestValidateID(t *testing.T) {
	if err := validateID("../etc"); err == nil {
		t.Fatal("expected rejection")
	}
	if err := validateID("redis"); err != nil {
		t.Fatal(err)
	}
}

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
