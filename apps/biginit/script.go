package main

import (
	"fmt"
	"os"
	"os/exec"
)

func startInitScript(path string) error {
	cmd := exec.Command(path, "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	return nil
}

func startShellScript(path string) error {
	// Match BusyBox rcS: subshell, reset traps, set positional $1=start, source script.
	cmd := exec.Command("sh", "-c", fmt.Sprintf(
		`( trap - INT QUIT TSTP; set start; . %q )`,
		path,
	))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	return nil
}
