package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverScripts(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, filepath.Join(dir, "S05-cron"), "#!/bin/sh\n")
	writeExec(t, filepath.Join(dir, "S10-mount"), "#!/bin/sh\n")
	writeExec(t, filepath.Join(dir, "S90-dropbear"), "#!/bin/sh\n")
	os.WriteFile(filepath.Join(dir, "rcS"), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(dir, "S99-not-exec"), []byte("#!/bin/sh\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "not-a-script"), []byte("#!/bin/sh\n"), 0o755)

	scripts, err := discoverScripts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 3 {
		t.Fatalf("got %d scripts, want 3: %+v", len(scripts), scripts)
	}
	if scripts[0].id != "cron" || scripts[1].id != "mount" || scripts[2].id != "dropbear" {
		t.Fatalf("unexpected order/ids: %+v", scripts)
	}
}

func TestMergeConfig(t *testing.T) {
	defaults := Config{Services: []ServiceConfig{
		{Name: "cron", Autostart: true, Retries: 0},
		{Name: "dropbear", Autostart: true, Retries: 0},
	}}
	user := Config{Services: []ServiceConfig{
		{Name: "dropbear", Autostart: false, Retries: 3},
	}}

	got := mergeConfig(defaults, user)
	cron, ok := got.lookup("cron")
	if !ok || !cron.Autostart || cron.Retries != 0 {
		t.Fatalf("cron = %+v", cron)
	}
	dropbear, ok := got.lookup("dropbear")
	if !ok || dropbear.Autostart || dropbear.Retries != 3 {
		t.Fatalf("dropbear = %+v", dropbear)
	}
}

func TestReadConfigFilePartialFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "biginit.yaml")
	content := `services:
- name: dropbear
  retries: 3
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := readConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	svc, ok := cfg.lookup("dropbear")
	if !ok {
		t.Fatal("dropbear not found")
	}
	if !svc.Autostart {
		t.Fatal("autostart should stay default true when omitted")
	}
	if svc.Retries != 3 {
		t.Fatalf("retries = %d, want 3", svc.Retries)
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "biginit.yaml")
	defaultsPath := filepath.Join(dir, "biginit.yaml.defaults")
	initDir := t.TempDir()
	writeExec(t, filepath.Join(initDir, "S90-dropbear"), "#!/bin/sh\n")

	scripts, err := discoverScripts(initDir)
	if err != nil {
		t.Fatal(err)
	}

	r := &runner{
		configPath:   configPath,
		defaultsPath: defaultsPath,
		scripts:      scripts,
	}
	if err := r.loadOrCreateConfig(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config not created: %v", err)
	}
	if _, err := os.Stat(defaultsPath); err != nil {
		t.Fatalf("defaults not created: %v", err)
	}

	svc, ok := r.config.lookup("dropbear")
	if !ok || !svc.Autostart || svc.Retries != 0 {
		t.Fatalf("config = %+v", svc)
	}
}

func TestServiceConfigBeforeLoad(t *testing.T) {
	r := &runner{}
	cfg := r.serviceConfig("cron")
	if !cfg.Autostart || cfg.Retries != 0 {
		t.Fatalf("got %+v", cfg)
	}
}

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
