//go:build linux

// configure-ethernet brings up the first Ethernet interface with a static
// address on common club subnets, or falls back to DHCP. It then stays running
// and re-runs the same bring-up logic when the link/address is lost.
package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultConfigPath    = "/data/etc/configure-ethernet.conf"
	defaultPrimaryAddr   = "192.168.0.120"
	defaultSecondaryAddr = "192.168.1.120"
	defaultPrefixLen     = 24
	defaultConnTimeout   = 10 // seconds — health-check interval while up
	defaultBackoffTime   = 30 // seconds — wait between failed bring-up attempts
	pingCount            = 1
	pingTimeoutSec       = 2
	dhcpWait             = 5 * time.Second

	// Absolute paths: PID 1 / microinit often start with no PATH, and Go's
	// LookPath then searches only /usr/bin:/bin (not /sbin).
	ipBin       = "/sbin/ip"
	dhclientBin = "/sbin/dhclient"
	pingBin     = "/bin/ping"
)

type settings struct {
	primaryAddr          string
	secondaryAddr        string
	connectionTimeoutSec int
	backoffTimeSec       int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := runDaemon(ctx, defaultConfigPath); err != nil {
		fmt.Fprintf(os.Stderr, "configure-ethernet: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(ctx context.Context, configPath string) error {
	cfg, err := loadOrCreateConfig(configPath, defaultPrimaryAddr, defaultSecondaryAddr, defaultConnTimeout, defaultBackoffTime)
	if err != nil {
		return err
	}

	_ = runCmd(ipBin, "link", "set", "lo", "up")

	fmt.Printf(
		"configure-ethernet: daemon starting (CONNECTION_TIMEOUT=%ds BACKOFF_TIME=%ds)\n",
		cfg.connectionTimeoutSec,
		cfg.backoffTimeSec,
	)

	connTimeout := time.Duration(cfg.connectionTimeoutSec) * time.Second
	backoff := time.Duration(cfg.backoffTimeSec) * time.Second

	for {
		if ctx.Err() != nil {
			fmt.Println("configure-ethernet: shutting down")
			return nil
		}

		// Bring-up loop: same logic as the original oneshot start.
		for {
			if ctx.Err() != nil {
				fmt.Println("configure-ethernet: shutting down")
				return nil
			}
			if connect(cfg) {
				break
			}
			fmt.Printf("configure-ethernet: bring-up failed; retry in %ds\n", cfg.backoffTimeSec)
			if !sleepCtx(ctx, backoff) {
				fmt.Println("configure-ethernet: shutting down")
				return nil
			}
		}

		// Monitor until the connection is lost, then reconnect.
		for {
			if !sleepCtx(ctx, connTimeout) {
				fmt.Println("configure-ethernet: shutting down")
				return nil
			}
			iface, err := firstEthernetInterface()
			if err != nil || !connectionHealthy(iface) {
				fmt.Println("configure-ethernet: connection lost; reconnecting")
				break
			}
		}
	}
}

// connect tries PRIMARY, SECONDARY, then DHCP on the first Ethernet interface.
func connect(cfg settings) bool {
	iface, err := firstEthernetInterface()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configure-ethernet: %v\n", err)
		return false
	}

	fmt.Printf("configure-ethernet: using interface %s\n", iface)
	_ = runCmd("/bin/killall", "dhclient")

	if tryStatic(iface, cfg.primaryAddr) {
		fmt.Printf("configure-ethernet: static %s OK (gateway %s)\n", cfg.primaryAddr, gatewayFor(cfg.primaryAddr))
		return true
	}

	if tryStatic(iface, cfg.secondaryAddr) {
		fmt.Printf("configure-ethernet: static %s OK (gateway %s)\n", cfg.secondaryAddr, gatewayFor(cfg.secondaryAddr))
		return true
	}

	if tryDHCP(iface) {
		fmt.Println("configure-ethernet: DHCP OK")
		return true
	}

	fmt.Fprintf(os.Stderr, "configure-ethernet: failed to configure %s (static and DHCP)\n", iface)
	return false
}

func connectionHealthy(iface string) bool {
	if !ifaceLinkUp(iface) || !ifaceHasIPv4(iface) {
		return false
	}
	if gw, ok := defaultGateway(); ok {
		return pingHost(gw)
	}
	return true
}

func ifaceLinkUp(iface string) bool {
	out, err := exec.Command(ipBin, "link", "show", "dev", iface).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "state UP") || strings.Contains(string(out), ",UP")
}

func loadOrCreateConfig(path, defaultPrimary, defaultSecondary string, defaultTimeout, defaultBackoff int) (settings, error) {
	defaults := settings{
		primaryAddr:          defaultPrimary,
		secondaryAddr:        defaultSecondary,
		connectionTimeoutSec: defaultTimeout,
		backoffTimeSec:       defaultBackoff,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return settings{}, fmt.Errorf("read %s: %w", path, err)
		}
		if writeErr := writeConfig(path, defaults); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot write %s: %v\n", path, writeErr)
		}
		return defaults, nil
	}

	return parseConfig(string(data), defaults), nil
}

func parseConfig(text string, defaults settings) settings {
	cfg := defaults

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
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		switch strings.ToUpper(key) {
		case "PRIMARY", "PRIMARY_ADDRESS", "ADDRESS":
			if net.ParseIP(value) != nil {
				cfg.primaryAddr = value
			}
		case "SECONDARY", "SECONDARY_ADDRESS", "FALLBACK", "FALLBACK_ADDRESS":
			if net.ParseIP(value) != nil {
				cfg.secondaryAddr = value
			}
		case "CONNECTION_TIMEOUT":
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				cfg.connectionTimeoutSec = n
			}
		case "BACKOFF_TIME":
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				cfg.backoffTimeSec = n
			}
		}
	}
	return cfg
}

func writeConfig(path string, cfg settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# configure-ethernet static addresses (edit to match club subnet)
PRIMARY=%s
SECONDARY=%s
# Seconds between health checks while connected
CONNECTION_TIMEOUT=%d
# Seconds to wait after a failed bring-up before retrying
BACKOFF_TIME=%d
`, cfg.primaryAddr, cfg.secondaryAddr, cfg.connectionTimeoutSec, cfg.backoffTimeSec)

	return os.WriteFile(path, []byte(content), 0o644)
}

func firstEthernetInterface() (string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", fmt.Errorf("list network interfaces: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, ent := range entries {
		name := ent.Name()
		if name == "lo" {
			continue
		}
		if isWireless(name) {
			continue
		}
		names = append(names, name)
	}

	if len(names) == 0 {
		return "", fmt.Errorf("no Ethernet interface found")
	}

	sort.Strings(names)
	return names[0], nil
}

func isWireless(iface string) bool {
	_, err := os.Stat(filepath.Join("/sys/class/net", iface, "wireless"))
	return err == nil
}

func tryStatic(iface, addr string) bool {
	gw := gatewayFor(addr)
	if gw == "" {
		return false
	}

	if err := configureStatic(iface, addr); err != nil {
		fmt.Fprintf(os.Stderr, "configure-ethernet: static %s on %s: %v\n", addr, iface, err)
		return false
	}

	if pingHost(gw) {
		return true
	}

	fmt.Fprintf(os.Stderr, "configure-ethernet: no reply from gateway %s\n", gw)
	return false
}

func configureStatic(iface, addr string) error {
	if err := runCmd(ipBin, "link", "set", "dev", iface, "up"); err != nil {
		return err
	}
	if err := runCmd(ipBin, "addr", "flush", "dev", iface); err != nil {
		return err
	}
	cidr := fmt.Sprintf("%s/%d", addr, defaultPrefixLen)
	return runCmd(ipBin, "addr", "add", cidr, "dev", iface)
}

func pingHost(host string) bool {
	err := runCmd(pingBin, "-c", fmt.Sprint(pingCount), "-W", fmt.Sprint(pingTimeoutSec), host)
	return err == nil
}

func tryDHCP(iface string) bool {
	_ = runCmd(ipBin, "addr", "flush", "dev", iface)
	_ = runCmd(ipBin, "link", "set", "dev", iface, "up")

	if err := runCmd(dhclientBin, iface); err != nil {
		fmt.Fprintf(os.Stderr, "configure-ethernet: dhclient on %s: %v\n", iface, err)
		return false
	}

	time.Sleep(dhcpWait)

	if !ifaceHasIPv4(iface) {
		fmt.Fprintf(os.Stderr, "configure-ethernet: no IPv4 address on %s after DHCP\n", iface)
		return false
	}

	if gw, ok := defaultGateway(); ok && pingHost(gw) {
		return true
	}

	// Lease without a pingable default route is still usable on some networks.
	return ifaceHasIPv4(iface)
}

func ifaceHasIPv4(iface string) bool {
	out, err := exec.Command(ipBin, "-4", "addr", "show", "dev", iface).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "inet ")
}

func defaultGateway() (string, bool) {
	out, err := exec.Command(ipBin, "route", "show", "default").Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "via" && i+1 < len(fields) {
				return fields[i+1], true
			}
		}
	}
	return "", false
}

func gatewayFor(addr string) string {
	ip := net.ParseIP(addr)
	if ip == nil {
		return ""
	}
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	ip[3] = 1
	return ip.String()
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
