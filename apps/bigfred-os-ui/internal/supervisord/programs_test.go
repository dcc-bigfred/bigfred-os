package supervisord

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListMergesStatus(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "supervisord.conf")
	if err := os.WriteFile(conf, []byte(sampleConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeCtl := filepath.Join(dir, "supervisorctl")
	// supervisord reports group members as "group:program".
	script := `#!/bin/sh
while [ "$1" = "-c" ]; do shift; shift; done
case "$1" in
status)
  echo "infra:redis RUNNING pid 99"
  echo "loco:scripts-executor STOPPED"
  ;;
esac
`
	if err := os.WriteFile(fakeCtl, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldBin := supervisorctlBin
	supervisorctlBin = fakeCtl
	t.Cleanup(func() { supervisorctlBin = oldBin })

	list, err := List(conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list: %+v", list)
	}
	byName := map[string]Program{}
	for _, p := range list {
		byName[p.Name] = p
	}
	if byName["redis"].Status != "RUNNING" || byName["redis"].PID != 99 {
		t.Fatalf("redis: %+v", byName["redis"])
	}
	if byName["scripts-executor"].Status != "STOPPED" {
		t.Fatalf("executor: %+v", byName["scripts-executor"])
	}
}

func TestControlStart(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "supervisord.conf")
	if err := os.WriteFile(conf, []byte(sampleConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeCtl := filepath.Join(dir, "supervisorctl")
	argsFile := filepath.Join(dir, "args")
	script := `#!/bin/sh
while [ "$1" = "-c" ]; do shift; shift; done
case "$1" in
start|stop|restart) echo "$1 $2" > "` + argsFile + `"; exit 0 ;;
esac
exit 1
`
	if err := os.WriteFile(fakeCtl, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldBin := supervisorctlBin
	supervisorctlBin = fakeCtl
	t.Cleanup(func() { supervisorctlBin = oldBin })

	if err := Control(conf, "redis", "start"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "start infra:redis" {
		t.Fatalf("supervisorctl args: %q", string(got))
	}
	if err := Control(conf, "missing", "start"); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
