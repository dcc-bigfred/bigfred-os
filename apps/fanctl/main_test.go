package main

import (
	"os"
	"path/filepath"
	"testing"
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
