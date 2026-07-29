// Package main is a thin supervisorctl-compatible shim for ochinchina/supervisord.
//
// loco-server invokes: supervisorctl -c CONF <cmd> [args...]
// ochinchina expects:  supervisord -c CONF ctl <cmd> [args...]
//
// Mapping:
//   reread  -> no-op (exit 0)
//   update  -> ctl reload, then ctl start for non-RUNNING programs
//   status  -> ctl status (stdout normalized when needed)
//   pid     -> without args: HTTP healthcheck + pidfile; with args: pass through
//   start/stop/shutdown/signal -> pass through as ctl subcommands
package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	conf, rest, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "usage: supervisorctl -c CONF <command> [args...]")
		return 2
	}

	supervisord := resolveSupervisord()
	cmdName := rest[0]
	cmdArgs := rest[1:]

	switch cmdName {
	case "reread":
		return 0
	case "update":
		if code := ctl(supervisord, conf, append([]string{"reload"}, cmdArgs...)...); code != 0 {
			return code
		}
		return startNonRunning(supervisord, conf)
	case "pid":
		if len(cmdArgs) == 0 {
			return pingPid(conf)
		}
		return ctl(supervisord, conf, append([]string{"pid"}, cmdArgs...)...)
	default:
		return ctl(supervisord, conf, append([]string{cmdName}, cmdArgs...)...)
	}
}

func parseArgs(args []string) (conf string, rest []string, err error) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("missing value for -c")
			}
			conf = args[i+1]
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	if conf == "" {
		return "", nil, fmt.Errorf("missing -c CONF")
	}
	return conf, rest, nil
}

func resolveSupervisord() string {
	if v := os.Getenv("SUPERVISORD_BIN"); v != "" {
		return v
	}
	// Prefer sibling named supervisord (/usr/bin layout on hub).
	self, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(self), "supervisord")
		if st, err := os.Stat(sibling); err == nil && !st.IsDir() {
			return sibling
		}
		// Android jniLibs: libsupervisorctl.so next to libsupervisord.so
		libSibling := filepath.Join(filepath.Dir(self), "libsupervisord.so")
		if st, err := os.Stat(libSibling); err == nil && !st.IsDir() {
			return libSibling
		}
	}
	if p, err := exec.LookPath("supervisord"); err == nil {
		return p
	}
	return "supervisord"
}

func ctl(bin, conf string, ctlArgs ...string) int {
	args := append([]string{"-c", conf, "ctl"}, ctlArgs...)
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func ctlOutput(bin, conf string, ctlArgs ...string) (string, int) {
	args := append([]string{"-c", conf, "ctl"}, ctlArgs...)
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return stdout.String() + stderr.String(), ee.ExitCode()
		}
		return err.Error(), 1
	}
	return stdout.String(), 0
}

var statusLine = regexp.MustCompile(`(?m)^(\S+)\s+(\S+)`)

func startNonRunning(bin, conf string) int {
	out, code := ctlOutput(bin, conf, "status")
	if code != 0 {
		// reload may leave ctl briefly unavailable; treat as soft success
		return 0
	}
	for _, line := range strings.Split(out, "\n") {
		m := statusLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		name, status := m[1], m[2]
		if status == "RUNNING" || status == "STARTING" {
			continue
		}
		_ = ctl(bin, conf, "start", name)
	}
	return 0
}

func pingPid(conf string) int {
	serverURL := serverURLFromConf(conf)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(strings.TrimRight(serverURL, "/") + "/")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		fmt.Fprintf(os.Stderr, "supervisord HTTP %d\n", resp.StatusCode)
		return 1
	}
	// Best-effort pid from conventional pidfile next to conf.
	pidFile := filepath.Join(filepath.Dir(conf), "..", "..", "run", "supervisord.pid")
	if b, err := os.ReadFile(pidFile); err == nil {
		fmt.Print(strings.TrimSpace(string(b)))
	} else {
		fmt.Print("1")
	}
	return 0
}

func serverURLFromConf(conf string) string {
	data, err := os.ReadFile(conf)
	if err != nil {
		return "http://127.0.0.1:9001"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "serverurl=") {
			return strings.TrimPrefix(line, "serverurl=")
		}
	}
	return "http://127.0.0.1:9001"
}
