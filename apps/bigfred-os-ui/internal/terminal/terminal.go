package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
)

const (
	defaultShell = "/bin/sh"
	defaultCols  = 80
	defaultRows  = 24
)

// Process is a spawned shell attached to a PTY master.
type Process struct {
	Master *os.File
	Cmd    *exec.Cmd
}

// Spawn starts an interactive shell in a new PTY.
func Spawn(shell string, args []string, env []string, cols, rows uint16) (*Process, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = defaultShell
	}
	if len(args) == 0 {
		args = []string{"-l"}
	}
	if cols == 0 {
		cols = defaultCols
	}
	if rows == 0 {
		rows = defaultRows
	}

	cmd := exec.Command(shell, args...)
	cmd.Env = env

	master, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	return &Process{Master: master, Cmd: cmd}, nil
}

// Resize updates the PTY window size.
func Resize(f *os.File, cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return fmt.Errorf("invalid terminal size")
	}
	return pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols})
}

// DefaultEnv builds a minimal environment for an interactive shell session.
func DefaultEnv(username string) []string {
	user := strings.TrimSpace(username)
	if user == "" {
		user = "root"
	}
	home := "/root"
	if user != "root" {
		home = "/home/" + user
	}
	return []string{
		"TERM=xterm-256color",
		"USER=" + user,
		"LOGNAME=" + user,
		"HOME=" + home,
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
	}
}
