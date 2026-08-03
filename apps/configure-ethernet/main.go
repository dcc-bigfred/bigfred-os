//go:build linux

// configure-ethernet brings up the first Ethernet interface with a static
// address on common club subnets, or falls back to DHCP.
// Subcommand "check" reports whether the link still looks connected (cheap).
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultConfigPath    = "/data/etc/configure-ethernet.conf"
	defaultPrimaryAddr   = "192.168.0.120"
	defaultSecondaryAddr = "192.168.1.120"
	defaultPrefixLen     = 24
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
	primaryAddr   string
	secondaryAddr string
}

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up", "configure", "start":
		if err := runConfigure(defaultConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "configure-ethernet: %v\n", err)
			os.Exit(1)
		}
	case "check":
		if checkConnected() {
			os.Exit(0)
		}
		os.Exit(1)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "configure-ethernet: unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [up|check]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  up     bring up Ethernet (static / DHCP) and exit\n")
	fmt.Fprintf(os.Stderr, "  check  exit 0 if link+IPv4 look OK, else 1 (cheap)\n")
}

func runConfigure(configPath string) error {
	cfg, err := loadOrCreateConfig(configPath, defaultPrimaryAddr, defaultSecondaryAddr)
	if err != nil {
		return err
	}

	_ = runCmd(ipBin, "link", "set", "lo", "up")

	if connect(cfg) {
		return nil
	}
	return fmt.Errorf("failed to configure ethernet (static and DHCP)")
}

// checkConnected is a cheap liveness probe: ethernet iface is UP and has IPv4.
// No ping (that would be slower and flaky on quiet networks).
func checkConnected() bool {
	iface, err := firstEthernetInterface()
	if err != nil {
		return false
	}
	return ifaceLinkUp(iface) && ifaceHasIPv4(iface)
}

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

func ifaceLinkUp(iface string) bool {
	out, err := exec.Command(ipBin, "link", "show", "dev", iface).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "state UP") || strings.Contains(string(out), ",UP")
}

func loadOrCreateConfig(path, defaultPrimary, defaultSecondary string) (settings, error) {
	defaults := settings{
		primaryAddr:   defaultPrimary,
		secondaryAddr: defaultSecondary,
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
`, cfg.primaryAddr, cfg.secondaryAddr)

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
