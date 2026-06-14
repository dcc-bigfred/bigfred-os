//go:build linux

// configure-ethernet brings up the first Ethernet interface with a static
// address on common club subnets, or falls back to DHCP.
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
	defaultConfigPath   = "/data/etc/configure-ethernet.conf"
	defaultPrimaryAddr  = "192.168.0.120"
	defaultSecondaryAddr = "192.168.1.120"
	defaultPrefixLen    = 24
	pingCount           = 1
	pingTimeoutSec      = 2
	dhcpWait            = 5 * time.Second
)

type config struct {
	path          string
	primaryAddr   string
	secondaryAddr string
}

func main() {
	cfg := config{
		path:          defaultConfigPath,
		primaryAddr:   defaultPrimaryAddr,
		secondaryAddr: defaultSecondaryAddr,
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "configure-ethernet: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	primary, secondary, err := loadOrCreateConfig(cfg.path, cfg.primaryAddr, cfg.secondaryAddr)
	if err != nil {
		return err
	}

	iface, err := firstEthernetInterface()
	if err != nil {
		return err
	}

	fmt.Printf("configure-ethernet: using interface %s\n", iface)

	if tryStatic(iface, primary) {
		fmt.Printf("configure-ethernet: static %s OK (gateway %s)\n", primary, gatewayFor(primary))
		return nil
	}

	if tryStatic(iface, secondary) {
		fmt.Printf("configure-ethernet: static %s OK (gateway %s)\n", secondary, gatewayFor(secondary))
		return nil
	}

	if tryDHCP(iface) {
		fmt.Println("configure-ethernet: DHCP OK")
		return nil
	}

	return fmt.Errorf("failed to configure %s (static and DHCP)", iface)
}

func loadOrCreateConfig(path, defaultPrimary, defaultSecondary string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("read %s: %w", path, err)
		}
		if writeErr := writeConfig(path, defaultPrimary, defaultSecondary); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot write %s: %v\n", path, writeErr)
		}
		return defaultPrimary, defaultSecondary, nil
	}

	primary, secondary := parseConfig(string(data), defaultPrimary, defaultSecondary)
	return primary, secondary, nil
}

func parseConfig(text, defaultPrimary, defaultSecondary string) (string, string) {
	primary := defaultPrimary
	secondary := defaultSecondary

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
				primary = value
			}
		case "SECONDARY", "SECONDARY_ADDRESS", "FALLBACK", "FALLBACK_ADDRESS":
			if net.ParseIP(value) != nil {
				secondary = value
			}
		}
	}
	return primary, secondary
}

func writeConfig(path, primary, secondary string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# configure-ethernet static addresses (edit to match club subnet)
PRIMARY=%s
SECONDARY=%s
`, primary, secondary)

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
	if err := runCmd("ip", "link", "set", "dev", iface, "up"); err != nil {
		return err
	}
	if err := runCmd("ip", "addr", "flush", "dev", iface); err != nil {
		return err
	}
	cidr := fmt.Sprintf("%s/%d", addr, defaultPrefixLen)
	return runCmd("ip", "addr", "add", cidr, "dev", iface)
}

func pingHost(host string) bool {
	err := runCmd("ping", "-c", fmt.Sprint(pingCount), "-W", fmt.Sprint(pingTimeoutSec), host)
	return err == nil
}

func tryDHCP(iface string) bool {
	_ = runCmd("ip", "addr", "flush", "dev", iface)
	_ = runCmd("ip", "link", "set", "dev", iface, "up")

	if err := runCmd("dhclient", iface); err != nil {
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
	out, err := exec.Command("ip", "-4", "addr", "show", "dev", iface).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "inet ")
}

func defaultGateway() (string, bool) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
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
