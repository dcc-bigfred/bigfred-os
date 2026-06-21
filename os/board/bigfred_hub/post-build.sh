#!/bin/sh
# Post-build: hub layout on target rootfs (before image assembly)

set -e

HUB="${BR2_EXTERNAL_BIGFRED_HUB_PATH:-$(dirname "$0")/../..}"

# Mount point for RW data partition (ext4 on nvme0n1p3)
mkdir -p "${TARGET_DIR}/data"

# Persistent log and application state directories (on /data at runtime)
mkdir -p "${TARGET_DIR}/data/sqlite"
mkdir -p "${TARGET_DIR}/data/redis"
mkdir -p "${TARGET_DIR}/data/alloy"
mkdir -p "${TARGET_DIR}/data/opt/grafana/data"
mkdir -p "${TARGET_DIR}/data/opt/grafana/log"
mkdir -p "${TARGET_DIR}/data/opt/grafana/plugins"
mkdir -p "${TARGET_DIR}/data/opt/victoriametrics"
mkdir -p "${TARGET_DIR}/data/logs/bigfred"
mkdir -p "${TARGET_DIR}/data/logs/redis"
mkdir -p "${TARGET_DIR}/data/logs/alloy"
mkdir -p "${TARGET_DIR}/data/etc"

# Placeholder for BigFred (installed separately by operator)
mkdir -p "${TARGET_DIR}/usr/share/bigfred/web"

# Install hub Go binaries (make -C apps build → apps/.bin/)
APPS_BIN="${HUB}/../apps/.bin"
if [ -d "${APPS_BIN}" ]; then
	installed=0
	for bin in "${APPS_BIN}"/*; do
		[ -f "$bin" ] || continue
		[ -x "$bin" ] || chmod 755 "$bin"
		name=$(basename "$bin")
		install -D -m 0755 "$bin" "${TARGET_DIR}/usr/sbin/${name}"
		installed=$((installed + 1))
	done
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

# bigfred-os-ui seed (copied to /data/etc on first boot by S10-mount)
if [ -f "${HUB}/board/bigfred_hub/bigfred-os-ui.conf" ]; then
	mkdir -p "${TARGET_DIR}/etc/bigfred"
	install -m 0644 "${HUB}/board/bigfred_hub/bigfred-os-ui.conf" \
		"${TARGET_DIR}/etc/bigfred/bigfred-os-ui.conf"
fi

# fanctl seed (copied to /data/etc on first boot by S10-mount)
if [ -f "${HUB}/board/bigfred_hub/fanctl.conf" ]; then
	mkdir -p "${TARGET_DIR}/etc/bigfred"
	install -m 0644 "${HUB}/board/bigfred_hub/fanctl.conf" \
		"${TARGET_DIR}/etc/bigfred/fanctl.conf"
fi

exit 0
