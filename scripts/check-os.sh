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
grep -q '^BR2_PACKAGE_LINUX_PREBUILT=y' "$defconfig" || {
	echo "defconfig: missing BR2_PACKAGE_LINUX_PREBUILT=y" >&2
	fail=1
}
if grep -q '^BR2_LINUX_KERNEL=y' "$defconfig"; then
	echo "defconfig: BR2_LINUX_KERNEL must be unset (prebuilt kernel)" >&2
	fail=1
fi
hashf="$OS/package/linux-prebuilt/linux-prebuilt.hash"
ver="$(sed -n 's/^LINUX_PREBUILT_VERSION = //p' "$OS/package/linux-prebuilt/linux-prebuilt.mk" | head -1)"
asset="bigfred-kernel-rpi5-v${ver}.tar.xz"
if [[ ! -f "$hashf" ]] || ! grep -qE "^sha256[[:space:]]+[0-9a-f]{64}[[:space:]]+${asset}\$" "$hashf"; then
	echo "linux-prebuilt.hash: must pin sha256 of ${asset}" >&2
	fail=1
fi
sh_n "$OS/board/bigfred_hub/post-image.sh"
sh_n "$ROOT/scripts/sync-kernel.sh"
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

grep -q 'BIGFRED_FORCE_REDOWNLOAD' "$OS/package/bigfred/bigfred.mk" || {
	echo "bigfred.mk: missing BIGFRED_FORCE_REDOWNLOAD (tip refs must re-fetch)" >&2
	fail=1
}
grep -q 'binaries-arm64' "$OS/package/bigfred/fetch.sh" || {
	echo "bigfred fetch.sh: ARTIFACT must be binaries-arm64" >&2
	fail=1
}
grep -q 'bigfred-dirclean' "$OS/Makefile" || {
	echo "os/Makefile: image target must run bigfred-dirclean before build" >&2
	fail=1
}
grep -q 'chown root:bigfred "$DATA_ROOT/run"' "$OS/overlays/etc/microinit/early-boot.sh" || {
	echo "early-boot.sh: /data/run must be root:bigfred (cap-stripped root services)" >&2
	fail=1
}
grep -q 'chmod 0770 "$DATA_ROOT/run"' "$OS/overlays/etc/microinit/early-boot.sh" || {
	echo "early-boot.sh: /data/run must be mode 0770 (not 0750 bigfred-only)" >&2
	fail=1
}
grep -q 'BR2_TARGET_GENERIC_HOSTNAME="bigfred"' "$defconfig" || {
	echo "defconfig: BR2_TARGET_GENERIC_HOSTNAME must be bigfred (dcc-bus mDNS host)" >&2
	fail=1
}
grep -q '"host": "bigfred"' "$OS/overlays/etc/microdns/microdns.json" || {
	echo "microdns.json overlay: dccBus.host must be bigfred" >&2
	fail=1
}
if [[ "$fail" -ne 0 ]]; then
	echo "check-os: FAIL" >&2
	exit 1
fi
echo "check-os: OK"
