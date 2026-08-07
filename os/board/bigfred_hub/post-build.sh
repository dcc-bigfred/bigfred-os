#!/bin/sh
# Post-build: hub layout on target rootfs (before image assembly)

set -e

HUB="${BR2_EXTERNAL_BIGFRED_HUB_PATH:-$(dirname "$0")/../..}"

# Mount point for RW data partition (ext4 LABEL=bigfred-data on SD or NVMe)
mkdir -p "${TARGET_DIR}/data"

# Persistent log and application state directories (on /data at runtime).
# Modes here are placeholders; early-boot.sh applies 0750 + service chown
# after users.table accounts exist (makeusers runs after post-build).
mkdir -p "${TARGET_DIR}/data/var/db/redis"
mkdir -p "${TARGET_DIR}/data/var/db/bigfred"
mkdir -p "${TARGET_DIR}/data/var/lib/alloy"
mkdir -p "${TARGET_DIR}/data/var/lib/victoriametrics"
mkdir -p "${TARGET_DIR}/data/var/lib/grafana/data"
mkdir -p "${TARGET_DIR}/data/var/lib/grafana/plugins"
# bigfred $HOME (tmpfs mounted at early-boot on RO root)
mkdir -p "${TARGET_DIR}/home/bigfred"
# Override path for BigFred binary (/usr/bin/bigfred prefers this over /opt)
mkdir -p "${TARGET_DIR}/data/opt/bigfred/bin"
mkdir -p "${TARGET_DIR}/data/logs/bigfred"
mkdir -p "${TARGET_DIR}/data/logs/redis"
mkdir -p "${TARGET_DIR}/data/logs/alloy"
mkdir -p "${TARGET_DIR}/data/logs/grafana"
mkdir -p "${TARGET_DIR}/data/etc"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/infra"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/dcc-bus"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/os"
# Persistent root SSH (bound over /root/.ssh at early-boot)
mkdir -p "${TARGET_DIR}/data/root/.ssh"
chmod 700 "${TARGET_DIR}/data/root/.ssh"
# Mount point on RO rootfs for the .ssh bind
mkdir -p "${TARGET_DIR}/root/.ssh"
chmod 700 "${TARGET_DIR}/root/.ssh"

# Placeholder for BigFred (installed separately by operator)
mkdir -p "${TARGET_DIR}/usr/share/bigfred/web"

# Install hub app binaries (make -C apps build → apps/.bin/)
APPS_BIN="${HUB}/../apps/.bin"
if [ -d "${APPS_BIN}" ]; then
	installed=0
	for bin in "${APPS_BIN}"/*; do
		[ -f "$bin" ] || continue
		case "$(basename "$bin")" in
			bigfred-os-ui-*)
				# Installed below as /usr/sbin/bigfred-os-ui
				continue
				;;
			biginit)
				# Replaced by microinit (package/microinit)
				continue
				;;
			configure-ethernet|configure-dhcp)
				# Replaced by micronet (package/micronet)
				continue
				;;
		esac
		[ -x "$bin" ] || chmod 755 "$bin"
		name=$(basename "$bin")
		install -D -m 0755 "$bin" "${TARGET_DIR}/usr/sbin/${name}"
		installed=$((installed + 1))
	done
	UI_TARGET="${APPS_BIN}/bigfred-os-ui-linux-arm64"
	if [ -f "${UI_TARGET}" ]; then
		install -D -m 0755 "${UI_TARGET}" "${TARGET_DIR}/usr/sbin/bigfred-os-ui"
		installed=$((installed + 1))
	fi
	if [ "$installed" -eq 0 ]; then
		echo "warning: ${APPS_BIN} is empty — run: make -C apps build" >&2
	fi
else
	echo "warning: ${APPS_BIN} missing — run: make -C apps build" >&2
fi

# Default network config template (edit per club)
if [ -f "${HUB}/board/bigfred_hub/network.conf" ] && \
   [ ! -f "${TARGET_DIR}/etc/bigfred/network.conf" ]; then
	mkdir -p "${TARGET_DIR}/etc/bigfred"
	install -m 0644 "${HUB}/board/bigfred_hub/network.conf" \
		"${TARGET_DIR}/etc/bigfred/network.conf"
fi

# bigfred-os-ui seed (copied to /data/etc on first boot by early-boot.sh)
if [ -f "${HUB}/board/bigfred_hub/bigfred-os-ui.conf" ]; then
	mkdir -p "${TARGET_DIR}/etc/bigfred"
	install -m 0644 "${HUB}/board/bigfred_hub/bigfred-os-ui.conf" \
		"${TARGET_DIR}/etc/bigfred/bigfred-os-ui.conf"
fi

# Default timezone (Europe/Warsaw); operator override lives on /data via early-boot bind.
if [ -e "${TARGET_DIR}/usr/share/zoneinfo/Europe/Warsaw" ]; then
	ln -sfn /usr/share/zoneinfo/Europe/Warsaw "${TARGET_DIR}/etc/localtime"
	printf 'Europe/Warsaw\n' > "${TARGET_DIR}/etc/timezone"
fi

# Skip leftover biginit binary if present in apps/.bin from older trees
rm -f "${TARGET_DIR}/usr/sbin/biginit"

# Build-time OS identity: git commit of the bigfred-os tree.
# prepare-nvme refuses to run unless /var/lib/bigfred exists (marker below).
REPO_ROOT="$(cd "${HUB}/.." && pwd)"
COMMIT="unknown"
if command -v git >/dev/null 2>&1 && [ -d "${REPO_ROOT}/.git" ]; then
	COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || true)"
	[ -n "${COMMIT}" ] || COMMIT="unknown"
elif [ -n "${GITHUB_SHA:-}" ]; then
	COMMIT="${GITHUB_SHA}"
fi
mkdir -p "${TARGET_DIR}/usr/lib/bigfred/version"
printf '%s\n' "${COMMIT}" > "${TARGET_DIR}/usr/lib/bigfred/version/commit"
chmod 0644 "${TARGET_DIR}/usr/lib/bigfred/version/commit"
mkdir -p "${TARGET_DIR}/var/lib/bigfred"
echo "bigfred-os commit: ${COMMIT}"

exit 0
