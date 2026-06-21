// fanctl controls the Raspberry Pi 5 active cooler via thermal hwmon (§8.8).
package main

import (
	"bufio"
	"fmt"
	"log/syslog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConfigPath = "/data/etc/fanctl.conf"
	defaultFanPWM     = "/sys/class/thermal/cooling_device0/cur_state"
	defaultFanMax     = "/sys/class/thermal/cooling_device0/max_state"
	defaultTherm      = "/sys/class/thermal/thermal_zone0/temp"
	defaultInterval   = 5 * time.Second
	defaultCooldown   = 300 * time.Second
)

type paths struct {
	fanPWM string
	fanMax string
	therm  string
}

type fanConfig struct {
	path     string
	fanPWM   string
	fanMax   string
	therm    string
	interval time.Duration
	cooldown time.Duration
}

func defaultFanConfig() fanConfig {
	return fanConfig{
		path:     defaultConfigPath,
		fanPWM:   defaultFanPWM,
		fanMax:   defaultFanMax,
		therm:    defaultTherm,
		interval: defaultInterval,
		cooldown: defaultCooldown,
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := loadOrCreateConfig(defaultConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fanctl: %v\n", err)
		os.Exit(1)
	}
	cfg = applyEnvOverrides(cfg)

	p := paths{
		fanPWM: cfg.fanPWM,
		fanMax: cfg.fanMax,
		therm:  cfg.therm,
	}

	switch os.Args[1] {
	case "daemon":
		os.Exit(runDaemon(p, cfg))
	case "stop":
		os.Exit(runStop(p))
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s {daemon|stop}\n", os.Args[0])
}

func loadOrCreateConfig(path string) (fanConfig, error) {
	cfg := defaultFanConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return cfg, fmt.Errorf("read %s: %w", path, err)
		}
		if writeErr := writeConfig(path, cfg); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot write %s: %v\n", path, writeErr)
		}
		return cfg, nil
	}

	return parseConfig(string(data), cfg), nil
}

func parseConfig(text string, cfg fanConfig) fanConfig {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		switch key {
		case "COOLDOWN":
			if secs, err := strconv.Atoi(value); err == nil && secs >= 0 {
				cfg.cooldown = time.Duration(secs) * time.Second
			}
		case "INTERVAL":
			if secs, err := strconv.Atoi(value); err == nil && secs > 0 {
				cfg.interval = time.Duration(secs) * time.Second
			}
		case "FAN_PWM", "FANCTL_FAN_PWM":
			cfg.fanPWM = value
		case "FAN_MAX", "FANCTL_FAN_MAX":
			cfg.fanMax = value
		case "THERM", "FANCTL_THERM":
			cfg.therm = value
		}
	}
	return cfg
}

func writeConfig(path string, cfg fanConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# fanctl — Raspberry Pi 5 active cooler (hub spec §8.8)
# Edit on the device under /data/etc/ (persists across image updates).

COOLDOWN=%d
INTERVAL=%d

# Optional sysfs overrides (defaults work on Raspberry Pi 5):
# FAN_PWM=%s
# FAN_MAX=%s
# THERM=%s
`, int(cfg.cooldown.Seconds()), int(cfg.interval.Seconds()), cfg.fanPWM, cfg.fanMax, cfg.therm)

	return os.WriteFile(path, []byte(content), 0o644)
}

func applyEnvOverrides(cfg fanConfig) fanConfig {
	if v := os.Getenv("FANCTL_FAN_PWM"); v != "" {
		cfg.fanPWM = v
	}
	if v := os.Getenv("FANCTL_FAN_MAX"); v != "" {
		cfg.fanMax = v
	}
	if v := os.Getenv("FANCTL_THERM"); v != "" {
		cfg.therm = v
	}
	if v := os.Getenv("FANCTL_COOLDOWN"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			cfg.cooldown = time.Duration(secs) * time.Second
		}
	}
	if v := os.Getenv("FANCTL_INTERVAL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.interval = time.Duration(secs) * time.Second
		}
	}
	return cfg
}

func runDaemon(p paths, cfg fanConfig) int {
	log, err := syslog.New(syslog.LOG_DAEMON|syslog.LOG_INFO, "fanctl")
	if err != nil {
		log = nil
	}
	logf := func(format string, args ...any) {
		if log != nil {
			_ = log.Info(fmt.Sprintf(format, args...))
		}
	}

	if _, err := readTempC(p.therm); err != nil {
		logf("no thermal_zone0, exiting")
		return 0
	}

	logf("starting fanctl daemon (config=%s cooldown=%s interval=%s)",
		cfg.path, cfg.cooldown, cfg.interval)
	last := -1
	var fanOnSince time.Time
	fanOn := false

	for {
		t, err := readTempC(p.therm)
		if err != nil {
			t = 50
		}
		desired := fanLevelForTemp(t)
		lvl := applyCooldown(desired, last, fanOnSince, fanOn, cfg.cooldown)
		if lvl != last {
			if lvl > 0 && last <= 0 {
				fanOnSince = time.Now()
				fanOn = true
			}
			if lvl == 0 {
				fanOn = false
			}
			_ = setFanLevel(p, lvl)
			logf("temp=%dC fan_level=%d", t, lvl)
			last = lvl
		}
		time.Sleep(cfg.interval)
	}
}

func runStop(p paths) int {
	_ = setFanLevel(p, 0)
	return 0
}

func readTempC(therm string) (int, error) {
	b, err := os.ReadFile(therm)
	if err != nil {
		return 0, err
	}
	milli, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, err
	}
	return milli / 1000, nil
}

// setFanLevel sets PWM state: 0=OFF, 1=LOW, 2=MED, 3=HIGH.
func setFanLevel(p paths, level int) error {
	maxB, err := os.ReadFile(p.fanMax)
	if err != nil {
		return nil
	}
	f, err := os.OpenFile(p.fanPWM, os.O_WRONLY, 0)
	if err != nil {
		return nil
	}
	_ = f.Close()

	max, err := strconv.Atoi(strings.TrimSpace(string(maxB)))
	if err != nil || max <= 0 {
		return nil
	}

	var val int
	switch level {
	case 0:
		val = 0
	case 1:
		val = max / 3
	case 2:
		val = max * 2 / 3
	case 3:
		val = max
	default:
		val = 0
	}

	return os.WriteFile(p.fanPWM, []byte(strconv.Itoa(val)), 0o644)
}

func fanLevelForTemp(tempC int) int {
	switch {
	case tempC < 45:
		return 0
	case tempC < 60:
		return 1
	case tempC < 70:
		return 2
	default:
		return 3
	}
}

// applyCooldown keeps the fan running at the current level for at least cooldown
// after turn-on, preventing rapid on/off cycling when temperature hovers near
// the off threshold. Speed reductions (level > 0) are not affected.
func applyCooldown(desired, current int, fanOnSince time.Time, fanOn bool, cooldown time.Duration) int {
	if desired > 0 {
		return desired
	}
	if current <= 0 {
		return 0
	}
	if !fanOn || cooldown <= 0 {
		return 0
	}
	if time.Since(fanOnSince) < cooldown {
		return current
	}
	return 0
}
