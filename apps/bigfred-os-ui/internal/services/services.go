package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const DefaultInitDir = "/etc/init.d"

var (
	ErrInvalidID     = errors.New("invalid service id")
	ErrInvalidAction = errors.New("invalid action")
	ErrNotFound      = errors.New("service not found")
)

var scriptNameRe = regexp.MustCompile(`^S[0-9]{2}-(.+)$`)

// Service describes one SysV init script on the hub.
type Service struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Script  string `json:"script"`
	Running bool   `json:"running"`
}

// List scans initDir for S??-* scripts (BusyBox SysV style).
func List(initDir string) ([]Service, error) {
	entries, err := os.ReadDir(initDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Service
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if name == "rcS" {
			continue
		}
		m := scriptNameRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}

		id := m[1]
		script := filepath.Join(initDir, name)
		out = append(out, Service{
			ID:      id,
			Name:    displayName(id),
			Script:  script,
			Running: isRunning(id),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Control runs an init script action (start|stop|restart).
func Control(initDir, id, action string) error {
	if err := validateID(id); err != nil {
		return err
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return ErrInvalidAction
	}

	script, err := resolveScript(initDir, id)
	if err != nil {
		return err
	}

	ctx, cancel := execTimeout(30 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, script, action)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("%s %s failed", id, action)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func resolveScript(initDir, id string) (string, error) {
	entries, err := os.ReadDir(initDir)
	if err != nil {
		return "", err
	}
	for _, ent := range entries {
		m := scriptNameRe.FindStringSubmatch(ent.Name())
		if m == nil || m[1] != id {
			continue
		}
		script := filepath.Join(initDir, ent.Name())
		info, err := os.Stat(script)
		if err != nil {
			return "", err
		}
		if info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("service not executable")
		}
		return script, nil
	}
	return "", ErrNotFound
}

func validateID(id string) error {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		return ErrInvalidID
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			continue
		}
		return ErrInvalidID
	}
	return nil
}

func isRunning(id string) bool {
	pidPath := filepath.Join("/var/run", id+".pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	return processAlive(pid)
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func displayName(id string) string {
	id = strings.TrimSuffix(id, ".example")
	return strings.ReplaceAll(id, "-", " ")
}

var execTimeout = func(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
