package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFanLevelForTemp(t *testing.T) {
	tests := []struct {
		temp int
		want int
	}{
		{20, 0},
		{44, 0},
		{45, 1},
		{59, 1},
		{60, 2},
		{69, 2},
		{70, 3},
		{85, 3},
	}
	for _, tc := range tests {
		if got := fanLevelForTemp(tc.temp); got != tc.want {
			t.Errorf("fanLevelForTemp(%d) = %d, want %d", tc.temp, got, tc.want)
		}
	}
}

func TestReadTempC(t *testing.T) {
	dir := t.TempDir()
	therm := filepath.Join(dir, "temp")
	if err := os.WriteFile(therm, []byte("45678\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTempC(therm)
	if err != nil {
		t.Fatal(err)
	}
	if got != 45 {
		t.Fatalf("got %d, want 45", got)
	}
}

func TestParseConfig(t *testing.T) {
	cfg := defaultFanConfig()
	text := `# comment
COOLDOWN=120
INTERVAL=10
FAN_PWM=/tmp/pwm
FAN_MAX=/tmp/max
THERM=/tmp/therm
`
	got := parseConfig(text, cfg)
	if got.cooldown != 120*time.Second {
		t.Fatalf("cooldown = %s, want 120s", got.cooldown)
	}
	if got.interval != 10*time.Second {
		t.Fatalf("interval = %s, want 10s", got.interval)
	}
	if got.fanPWM != "/tmp/pwm" || got.fanMax != "/tmp/max" || got.therm != "/tmp/therm" {
		t.Fatalf("paths = %+v", got)
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fanctl.conf")

	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.cooldown != defaultCooldown {
		t.Fatalf("cooldown = %s, want %s", cfg.cooldown, defaultCooldown)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file created: %v", err)
	}

	if err := os.WriteFile(path, []byte("COOLDOWN=60\nINTERVAL=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = loadOrCreateConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.cooldown != 60*time.Second || cfg.interval != 2*time.Second {
		t.Fatalf("got cooldown=%s interval=%s", cfg.cooldown, cfg.interval)
	}
}

func TestApplyCooldown(t *testing.T) {
	cooldown := 300 * time.Second
	onSince := time.Now().Add(-60 * time.Second)

	tests := []struct {
		name     string
		desired  int
		current  int
		onSince  time.Time
		fanOn    bool
		cooldown time.Duration
		want     int
	}{
		{"off stays off", 0, 0, time.Time{}, false, cooldown, 0},
		{"turn on", 2, 0, time.Time{}, false, cooldown, 2},
		{"cooldown blocks off", 0, 2, onSince, true, cooldown, 2},
		{"cooldown expired allows off", 0, 2, time.Now().Add(-301 * time.Second), true, cooldown, 0},
		{"cooldown disabled allows off", 0, 2, onSince, true, 0, 0},
		{"speed reduction allowed during cooldown", 1, 3, onSince, true, cooldown, 1},
		{"speed increase allowed during cooldown", 3, 1, onSince, true, cooldown, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := applyCooldown(tc.desired, tc.current, tc.onSince, tc.fanOn, tc.cooldown)
			if got != tc.want {
				t.Errorf("applyCooldown() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSetFanLevel(t *testing.T) {
	dir := t.TempDir()
	maxPath := filepath.Join(dir, "max_state")
	pwmPath := filepath.Join(dir, "cur_state")
	if err := os.WriteFile(maxPath, []byte("9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pwmPath, []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := paths{fanPWM: pwmPath, fanMax: maxPath}
	if err := setFanLevel(p, 2); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(pwmPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "6" {
		t.Fatalf("cur_state = %q, want 6", b)
	}
}
