package terminal

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestSpawnRunsCommand(t *testing.T) {
	proc, err := Spawn("/bin/sh", []string{"-c", "echo hi; exit"}, nil, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Master.Close()

	done := make(chan struct{})
	go func() {
		_ = proc.Cmd.Wait()
		close(done)
	}()

	var out strings.Builder
	buf := make([]byte, 256)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			goto finished
		default:
		}
		n, err := proc.Master.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		out.Write(buf[:n])
	}
finished:

	if !strings.Contains(out.String(), "hi") {
		t.Fatalf("expected output to contain hi, got %q", out.String())
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shell did not exit")
	}
}

func TestResize(t *testing.T) {
	proc, err := Spawn("/bin/sh", []string{"-c", "sleep 1"}, nil, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Master.Close()
	}()

	if err := Resize(proc.Master, 120, 40); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultEnv(t *testing.T) {
	env := DefaultEnv("root")
	found := false
	for _, e := range env {
		if e == "USER=root" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected USER=root in %v", env)
	}
}
