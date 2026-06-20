package supervisord

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Ctl wraps supervisorctl for a single config file.
type Ctl struct {
	Bin        string
	ConfigPath string
}

// ProgramStatus is one row from supervisorctl status.
type ProgramStatus struct {
	Name   string
	Status string
	PID    int
}

var statusLinePattern = regexp.MustCompile(`^(\S+)\s+(\S+)(?:\s+pid\s+(\d+))?`)

func (c *Ctl) Status(ctx context.Context) ([]ProgramStatus, error) {
	out, err := c.run(ctx, "status")
	if err != nil {
		return nil, err
	}
	return parseStatusOutput(out), nil
}

func (c *Ctl) StartProgram(ctx context.Context, name string) error {
	_, err := c.run(ctx, "start", name)
	return err
}

func (c *Ctl) StopProgram(ctx context.Context, name string) error {
	_, err := c.run(ctx, "stop", name)
	return err
}

func (c *Ctl) RestartProgram(ctx context.Context, name string) error {
	_, err := c.run(ctx, "restart", name)
	return err
}

func (c *Ctl) run(ctx context.Context, args ...string) (string, error) {
	bin := c.Bin
	if bin == "" {
		bin = supervisorctlBin
	}
	cmdArgs := append([]string{"-c", c.ConfigPath}, args...)
	cmd := exec.CommandContext(ctx, bin, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("supervisorctl %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func parseStatusOutput(out string) []ProgramStatus {
	var rows []ProgramStatus
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := statusLinePattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		row := ProgramStatus{Name: m[1], Status: m[2]}
		if len(m) >= 4 && m[3] != "" {
			row.PID, _ = strconv.Atoi(m[3])
		}
		rows = append(rows, row)
	}
	return rows
}
