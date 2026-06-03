// fanctl controls the Raspberry Pi 5 active cooler via thermal hwmon (§8.8).
package main

import (
	"fmt"
	"log/syslog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultFanPWM  = "/sys/class/thermal/cooling_device0/cur_state"
	defaultFanMax  = "/sys/class/thermal/cooling_device0/max_state"
	defaultTherm   = "/sys/class/thermal/thermal_zone0/temp"
	defaultInterval = 5 * time.Second
)

type paths struct {
	fanPWM string
	fanMax string
	therm  string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	p := paths{
		fanPWM: envOr("FANCTL_FAN_PWM", defaultFanPWM),
		fanMax: envOr("FANCTL_FAN_MAX", defaultFanMax),
		therm:  envOr("FANCTL_THERM", defaultTherm),
	}

	switch os.Args[1] {
	case "daemon":
		os.Exit(runDaemon(p))
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runDaemon(p paths) int {
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

	logf("starting fanctl daemon")
	last := -1

	for {
		t, err := readTempC(p.therm)
		if err != nil {
			t = 50
		}
		lvl := fanLevelForTemp(t)
		if lvl != last {
			_ = setFanLevel(p, lvl)
			logf("temp=%dC fan_level=%d", t, lvl)
			last = lvl
		}
		time.Sleep(defaultInterval)
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
