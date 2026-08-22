#!/usr/bin/env bash
# Static checks for hub OS overlays and micronet packaging (no image build).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OS="$ROOT/os"
fail=0

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "error: $1 is required" >&2
		exit 1
	fi
}

need python3

sh_n() {
	local f="$1"
	if command -v dash >/dev/null 2>&1; then
		dash -n "$f" || { echo "dash -n failed: $f" >&2; fail=1; }
	else
		sh -n "$f" || { echo "sh -n failed: $f" >&2; fail=1; }
	fi
}

while IFS= read -r -d '' f; do
	sh_n "$f"
	if [[ ! -x "$f" ]]; then
		echo "not executable: $f" >&2
		fail=1
	fi
done < <(find "$OS/overlays/etc/init.d" -type f -print0 2>/dev/null)

sh_n "$OS/overlays/etc/microinit/early-boot.sh"
if [[ -f "$OS/overlays/etc/microinit/unmount.sh" ]]; then
	sh_n "$OS/overlays/etc/microinit/unmount.sh"
fi
while IFS= read -r -d '' f; do
	sh_n "$f"
done < <(find "$OS/board" -name 'post-build*.sh' -print0 2>/dev/null)
while IFS= read -r -d '' f; do
	sh_n "$f"
done < <(find "$OS/board" -path '*/post-build.d/*' -type f -print0 2>/dev/null)

while IFS= read -r -d '' f; do
	python3 -m json.tool "$f" >/dev/null || {
		echo "invalid JSON: $f" >&2
		fail=1
	}
done < <(find "$OS/overlays" -name '*.json' -print0)

cfg="$OS/package/Config.in"
defconfig="$OS/configs/bigfred_hub_rpi5_defconfig"
mk="$OS/package/micronet/micronet.mk"
fetch="$OS/package/micronet/fetch.sh"
net_json="$OS/overlays/etc/microinit.d/services/os/network.json"
net_init="$OS/overlays/etc/init.d/network"

grep -q 'select BR2_PACKAGE_DNSMASQ' "$cfg" || {
	echo "Config.in: missing select BR2_PACKAGE_DNSMASQ" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_DNSMASQ=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_DNSMASQ=y" >&2
	fail=1
}
grep -q 'MICRONET_DEPENDENCIES.*dnsmasq' "$mk" || {
	echo "micronet.mk: MICRONET_DEPENDENCIES must include dnsmasq" >&2
	fail=1
}
grep -q 'micronet-linux-arm64:bin/micronet' "$fetch" || {
	echo "fetch.sh: FILES must map micronet-linux-arm64" >&2
	fail=1
}
if grep -q 'configure-ethernet-linux-arm64' "$fetch" && ! grep -q 'micronet-linux-arm64' "$fetch"; then
	echo "fetch.sh: must not fall back to legacy-only artifacts" >&2
	fail=1
fi
grep -q 'micronet check' "$net_json" || {
	echo "network.json: liveness must be micronet check" >&2
	fail=1
}
grep -q '"restartCmd"' "$net_json" || {
	echo "network.json: missing restartCmd" >&2
	fail=1
}
grep -q '"daemon": true' "$net_json" || {
	echo "network.json: daemon must be true" >&2
	fail=1
}
grep -q 'dhclient' "$net_init" || {
	echo "init.d/network: stop must mention dhclient" >&2
	fail=1
}
grep -q 'teardown' "$net_init" || {
	echo "init.d/network: stop must call micronet teardown" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_BRCMFMAC_SDIO_FIRMWARE_RPI_WIFI=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_BRCMFMAC_SDIO_FIRMWARE_RPI_WIFI=y (Pi 5 wlan0)" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_BRCMFMAC_SDIO_FIRMWARE_RPI_BT=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_BRCMFMAC_SDIO_FIRMWARE_RPI_BT=y (Pi 5 hci0)" >&2
	fail=1
}
grep -q '^CONFIG_BRCMFMAC=m' "$OS/configs/linux-hub.fragment" || {
	echo "linux-hub.fragment: CONFIG_BRCMFMAC must be a module (firmware after rootfs)" >&2
	fail=1
}
grep -q 'prepare_dropbear_keys' "$OS/overlays/etc/init.d/dropbear" || {
	echo "init.d/dropbear: must create host-key dir (not rely on /var/run/dropbear)" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_OPENSSH=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_OPENSSH=y (sftp-server for scp)" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_OPENSSH_SERVER=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_OPENSSH_SERVER=y (installs sftp-server)" >&2
	fail=1
}
grep -q 'sftp-server' "$OS/board/bigfred_hub/post-build.d/15-openssh-sftp-only.sh" || {
	echo "post-build: must strip sshd, keep sftp-server (15-openssh-sftp-only.sh)" >&2
	fail=1
}
grep -q '^BR2_PACKAGE_NANO=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_NANO=y" >&2
	fail=1
}
grep -q '/etc/dropbear' "$OS/board/bigfred_hub/post-build.d/10-layout.sh" || {
	echo "10-layout.sh: must replace Buildroot /etc/dropbear -> /var/run/dropbear symlink" >&2
	fail=1
}
if grep -q '^BR2_PACKAGE_LINUX_FIRMWARE_BRCM_BCM43XXX=y' "$defconfig"; then
	echo "defconfig: BRCM_BCM43XXX conflicts with BRCMFMAC_SDIO_FIRMWARE_RPI_WIFI" >&2
	fail=1
fi

if [[ "$fail" -ne 0 ]]; then
	echo "check-os: FAIL" >&2
	exit 1
fi
echo "check-os: OK"
