//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig_defaults(t *testing.T) {
	defaults := settings{
		primaryAddr:   "192.168.0.120",
		secondaryAddr: "192.168.1.120",
	}
	cfg := parseConfig("", defaults)
	if cfg.primaryAddr != "192.168.0.120" || cfg.secondaryAddr != "192.168.1.120" {
		t.Fatalf("got primary=%q secondary=%q", cfg.primaryAddr, cfg.secondaryAddr)
	}
}

func TestParseConfig_overrides(t *testing.T) {
	text := `# club subnet
PRIMARY=10.0.0.50
SECONDARY=10.0.1.50
`
	defaults := settings{
		primaryAddr:   "192.168.0.120",
		secondaryAddr: "192.168.1.120",
	}
	cfg := parseConfig(text, defaults)
	if cfg.primaryAddr != "10.0.0.50" || cfg.secondaryAddr != "10.0.1.50" {
		t.Fatalf("got primary=%q secondary=%q", cfg.primaryAddr, cfg.secondaryAddr)
	}
}

func TestParseConfig_ignoresInvalidIP(t *testing.T) {
	text := "PRIMARY=not-an-ip\nSECONDARY=192.168.1.99\n"
	defaults := settings{
		primaryAddr:   "192.168.0.120",
		secondaryAddr: "192.168.1.120",
	}
	cfg := parseConfig(text, defaults)
	if cfg.primaryAddr != "192.168.0.120" || cfg.secondaryAddr != "192.168.1.99" {
		t.Fatalf("got primary=%q secondary=%q", cfg.primaryAddr, cfg.secondaryAddr)
	}
}

func TestGatewayFor(t *testing.T) {
	tests := map[string]string{
		"192.168.0.120": "192.168.0.1",
		"192.168.1.120": "192.168.1.1",
		"10.20.30.40":   "10.20.30.1",
	}
	for addr, want := range tests {
		if got := gatewayFor(addr); got != want {
			t.Fatalf("gatewayFor(%q) = %q, want %q", addr, got, want)
		}
	}
	if gatewayFor("bad") != "" {
		t.Fatal("expected empty gateway for invalid address")
	}
}

func TestLoadOrCreateConfig_createsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "configure-ethernet.conf")

	cfg, err := loadOrCreateConfig(path, "192.168.0.120", "192.168.1.120")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.primaryAddr != "192.168.0.120" || cfg.secondaryAddr != "192.168.1.120" {
		t.Fatalf("got primary=%q secondary=%q", cfg.primaryAddr, cfg.secondaryAddr)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{
		"PRIMARY=192.168.0.120",
		"SECONDARY=192.168.1.120",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, data)
		}
	}
}

func TestLoadOrCreateConfig_readsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "configure-ethernet.conf")
	content := "PRIMARY=172.16.0.8\nSECONDARY=172.16.1.8\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadOrCreateConfig(path, "192.168.0.120", "192.168.1.120")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.primaryAddr != "172.16.0.8" || cfg.secondaryAddr != "172.16.1.8" {
		t.Fatalf("got primary=%q secondary=%q", cfg.primaryAddr, cfg.secondaryAddr)
	}
}

func TestIsWireless(t *testing.T) {
	root := t.TempDir()
	eth := filepath.Join(root, "eth0")
	wlan := filepath.Join(root, "wlan0")

	if err := os.MkdirAll(filepath.Join(eth, "device"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wlan, "wireless"), 0o755); err != nil {
		t.Fatal(err)
	}

	if isWirelessAt(root, "eth0") {
		t.Fatal("eth0 should not be wireless")
	}
	if !isWirelessAt(root, "wlan0") {
		t.Fatal("wlan0 should be wireless")
	}
}

func isWirelessAt(root, iface string) bool {
	_, err := os.Stat(filepath.Join(root, iface, "wireless"))
	return err == nil
}
