#!/bin/sh
# BigFred OS early-boot for microinit (PID 1).
# Installed as /etc/microinit/early-boot.sh via rootfs overlay.
# Override on device: $DATA_DIR/etc/microinit/early-boot.sh
# Env: DATA_DIR, MICROINIT_LOGS_TTY, MICROINIT_INIT_LOGS_TTY, MICROINIT_CONSOLE
#
# Portable base (pseudo-FS + fstab) plus BigFred data root, seeding, shadow bind.
# Replaces the former SysV S00-mount script for the microinit boot path.
#
# Exit 0 on success. Non-zero aborts microinit boot when required.

set -eu

log() {
	echo "early-boot: $*" >&2
}

# Absolute DATA_DIR only; default /data.
case "${DATA_DIR:-}" in
/*) DATA_ROOT=$DATA_DIR ;;
*) DATA_ROOT=/data ;;
esac

is_mounted() {
	grep -q " $1 " /proc/mounts 2>/dev/null
}

mount_one() {
	_type=$1
	_dev=$2
	_dir=$3
	_opts=${4:-}
	if [ -r /proc/mounts ] && is_mounted "$_dir"; then
		return 0
	fi
	if command -v mountpoint >/dev/null 2>&1; then
		mountpoint -q "$_dir" 2>/dev/null && return 0
	fi
	mkdir -p "$_dir"
	if [ -n "$_opts" ]; then
		mount -t "$_type" -o "$_opts" "$_dev" "$_dir" || return 1
	else
		mount -t "$_type" "$_dev" "$_dir" || return 1
	fi
}

# --- essential pseudo filesystems (needed before fstab) ---
if ! [ -r /proc/mounts ]; then
	mkdir -p /proc
	mount -t proc proc /proc || true
fi

mount_one sysfs sysfs /sys || true
mount_one devtmpfs devtmpfs /dev "mode=0755,nosuid" || true
mkdir -p /dev/pts
mount_one devpts devpts /dev/pts "mode=0620,gid=5" || true
mount_one tmpfs tmpfs /run "mode=0755,nosuid,nodev" || true
mount_one tmpfs tmpfs /tmp "mode=1777,nosuid,nodev" || true

# --- fstab (incl. /data, tmpfs for /var/log, /var/run, …) ---
if [ -r /etc/fstab ]; then
	log "mount -a"
	mount -a || true
else
	log "no /etc/fstab; skipping mount -a"
fi

mkdir -p /var/log /var/run /run "$DATA_ROOT"

# --- /data partition fallbacks (hub default root only) ---
mount_data() {
	if is_mounted /data; then
		return 0
	fi
	if command -v findfs >/dev/null 2>&1; then
		_dev=$(findfs LABEL=bigfred-data 2>/dev/null || true)
		if [ -n "${_dev:-}" ] && [ -b "$_dev" ]; then
			mount -t ext4 "$_dev" /data && return 0
		fi
	fi
	for _dev in /dev/disk/by-label/bigfred-data /dev/mmcblk0p3 /dev/nvme0n1p3; do
		if [ -b "$_dev" ] || [ -e "$_dev" ]; then
			if mount -t ext4 "$_dev" /data 2>/dev/null; then
				return 0
			fi
		fi
	done
	# Unformatted partition — create once (empty TYPE only)
	for _dev in /dev/mmcblk0p3 /dev/nvme0n1p3; do
		if [ -b "$_dev" ]; then
			_fstype=$(blkid -o value -s TYPE "$_dev" 2>/dev/null || true)
			if [ -z "$_fstype" ]; then
				log "formatting $_dev as ext4 (LABEL=bigfred-data)"
				mkfs.ext4 -F -L bigfred-data "$_dev"
				mount -t ext4 "$_dev" /data && return 0
			fi
		fi
	done
	return 1
}

if [ "$DATA_ROOT" = /data ]; then
	if ! mount_data; then
		log "WARNING: could not mount /data — continuing with local directory"
		mkdir -p /data
	fi
else
	log "using custom data root $DATA_ROOT (skip partition mount)"
	mkdir -p "$DATA_ROOT"
fi

# --- directories ---
mkdir -p "$DATA_ROOT/etc" "$DATA_ROOT/logs" "$DATA_ROOT/sqlite" \
	"$DATA_ROOT/redis" "$DATA_ROOT/alloy"
mkdir -p "$DATA_ROOT/opt/bigfred/bin" "$DATA_ROOT/opt/grafana/data" \
	"$DATA_ROOT/opt/grafana/log" "$DATA_ROOT/opt/grafana/plugins" \
	"$DATA_ROOT/opt/victoriametrics"
mkdir -p "$DATA_ROOT/logs/bigfred" "$DATA_ROOT/logs/redis" "$DATA_ROOT/logs/alloy"
mkdir -p "$DATA_ROOT/etc/microinit"

# --- seed configs from image if missing ---
seed() {
	_src=$1
	_dst=$2
	_mode=${3:-}
	if [ -e "$_src" ] && [ ! -e "$_dst" ]; then
		mkdir -p "$(dirname "$_dst")"
		cp -a "$_src" "$_dst"
		if [ -n "$_mode" ]; then
			chmod "$_mode" "$_dst"
		fi
	fi
}

if [ -d /etc/bigfred ]; then
	for f in /etc/bigfred/*; do
		[ -e "$f" ] || continue
		seed "$f" "$DATA_ROOT/etc/$(basename "$f")" 640
	done
fi
seed /etc/redis/redis.conf "$DATA_ROOT/etc/redis.conf" 640
# Service list for microinit (PID 1); only seed if operator has not customized yet.
seed /etc/microinit/microinit.json "$DATA_ROOT/etc/microinit.json" 644
seed /etc/microinit/microinit.json "$DATA_ROOT/etc/microinit.json.example" 644

# --- persistent root password via bind-mounted shadow ---
if [ ! -f "$DATA_ROOT/etc/shadow" ] && [ -f /etc/shadow ]; then
	cp -a /etc/shadow "$DATA_ROOT/etc/shadow"
	chmod 600 "$DATA_ROOT/etc/shadow"
fi
if [ -f "$DATA_ROOT/etc/shadow" ]; then
	if [ "$DATA_ROOT" = /data ]; then
		if is_mounted /data && ! is_mounted /etc/shadow; then
			mount --bind "$DATA_ROOT/etc/shadow" /etc/shadow 2>/dev/null || true
		fi
	elif ! is_mounted /etc/shadow; then
		mount --bind "$DATA_ROOT/etc/shadow" /etc/shadow 2>/dev/null || true
	fi
fi

# Enforce read-only root (hub policy; ignore failure on RW systems)
mount -o remount,ro / 2>/dev/null || true

log "done (DATA_ROOT=$DATA_ROOT)"
exit 0
