#!/bin/sh
# BigFred OS late-shutdown unmount for microinit (PID 1).
# Installed as /etc/microinit/unmount.sh via rootfs overlay.
# Override on device: $DATA_DIR/etc/microinit/unmount.sh
# Env: DATA_DIR, MICROINIT_LOGS_TTY, MICROINIT_INIT_LOGS_TTY, MICROINIT_CONSOLE
#
# Reverse of early-boot + clean block-device shutdown:
#   1) drop /etc/shadow bind (holds /data busy)
#   2) remount RO + umount /data (persistent ext4 — SD p3 or NVMe)
#   3) remount root RO (flush journal if anything remounted RW at runtime)
#   4) umount -a -r for any remaining real mounts (skip virt/tmpfs)
#
# tmpfs (/tmp, /var/log, /var/run, /run) and proc/sys/dev are left alone —
# they hold no durable state and the kernel still needs them until reboot(2).
#
# Called after all supervised services are stopped. Failures are logged by
# microinit but do not block reboot.
#
# Exit 0 on success.

set -eu

log() {
	echo "unmount: $*" >&2
}

# Absolute DATA_DIR only; default /data.
case "${DATA_DIR:-}" in
/*) DATA_ROOT=$DATA_DIR ;;
*) DATA_ROOT=/data ;;
esac

is_mounted() {
	grep -q " $1 " /proc/mounts 2>/dev/null
}

try_umount() {
	_mp=$1
	umount "$_mp" 2>/dev/null && return 0
	umount -l "$_mp" 2>/dev/null && return 0
	return 1
}

sync || true

# --- 1) shadow bind must go before /data ---
if is_mounted /etc/shadow; then
	log "umount /etc/shadow"
	try_umount /etc/shadow || log "WARNING: could not umount /etc/shadow"
fi

# --- 2) persistent data root (mmcblk0p3 or NVMe after prepare-nvme) ---
if is_mounted "$DATA_ROOT"; then
	log "remount,ro $DATA_ROOT"
	mount -o remount,ro "$DATA_ROOT" 2>/dev/null || true
	log "umount $DATA_ROOT"
	try_umount "$DATA_ROOT" || log "WARNING: could not umount $DATA_ROOT"
fi

# --- 3) root: cannot umount while PID 1 runs; remount RO is the goal ---
log "remount,ro /"
mount -o remount,ro / 2>/dev/null || true

# --- 4) sweep any leftover real mounts (future fstab, accidental mounts) ---
# Exclude virt FS + tmpfs; -r remounts busy mounts RO instead of failing hard.
if command -v umount >/dev/null 2>&1; then
	log "umount -a -r (skip proc,sysfs,devtmpfs,devpts,tmpfs)"
	umount -a -r -t noproc,nosysfs,nodevtmpfs,nodevpts,notmpfs 2>/dev/null || true
fi

sync || true
log "done (DATA_ROOT=$DATA_ROOT)"
exit 0
