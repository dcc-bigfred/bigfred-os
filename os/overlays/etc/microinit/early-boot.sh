#!/bin/sh
# BigFred OS early-boot for microinit (PID 1).
# Installed as /etc/microinit/early-boot.sh via rootfs overlay.
# Override on device: $DATA_DIR/etc/microinit/early-boot.sh
# Env: DATA_DIR, MICROINIT_LOGS_TTY, MICROINIT_INIT_LOGS_TTY, MICROINIT_CONSOLE
#
# Portable base (pseudo-FS + fsck -y + fstab) plus BigFred data root, seeding,
# shadow and root SSH binds. Replaces the former SysV mount init script for
# the microinit boot path.
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

# True if $1 (block device path or symlink) is a source in /proc/mounts.
dev_is_mounted() {
	_want=$1
	_real=$(readlink -f "$_want" 2>/dev/null || echo "$_want")
	[ -r /proc/mounts ] || return 1
	while read -r _src _mp _fstype _opts _rest; do
		case "$_src" in
		\#* | '') continue ;;
		esac
		_sreal=$(readlink -f "$_src" 2>/dev/null || echo "$_src")
		if [ "$_src" = "$_want" ] || [ "$_src" = "$_real" ] || \
			[ "$_sreal" = "$_real" ] || [ "$_sreal" = "$_want" ]; then
			return 0
		fi
	done < /proc/mounts
	return 1
}

# Auto-repair one block device. Never aborts boot (fsck exit codes are noisy).
# Skips missing/non-block devices, unknown/empty TYPE, and already-mounted sources.
fsck_one() {
	_dev=$1
	[ -n "$_dev" ] || return 0
	case "$_dev" in
	LABEL=* | UUID=* | PARTUUID=* | PARTLABEL=*)
		if command -v findfs >/dev/null 2>&1; then
			_resolved=$(findfs "$_dev" 2>/dev/null || true)
			[ -n "${_resolved:-}" ] && _dev=$_resolved
		fi
		;;
	esac
	[ -b "$_dev" ] || return 0

	if dev_is_mounted "$_dev"; then
		log "fsck skip $_dev (already mounted)"
		return 0
	fi

	_fstype=
	if command -v blkid >/dev/null 2>&1; then
		_fstype=$(blkid -o value -s TYPE "$_dev" 2>/dev/null || true)
	fi
	case "$_fstype" in
	'' | swap | crypto_LUKS)
		return 0
		;;
	esac

	_rc=0
	if command -v fsck >/dev/null 2>&1; then
		log "fsck -y $_dev (TYPE=${_fstype:-unknown})"
		fsck -y -T "$_dev" || _rc=$?
	elif command -v e2fsck >/dev/null 2>&1; then
		case "$_fstype" in
		ext2 | ext3 | ext4)
			log "e2fsck -y $_dev"
			e2fsck -y "$_dev" || _rc=$?
			;;
		*)
			log "fsck skip $_dev (no fsck binary for TYPE=$_fstype)"
			return 0
			;;
		esac
	else
		log "fsck skip $_dev (no fsck/e2fsck on PATH)"
		return 0
	fi

	case $_rc in
	0) ;;
	1) log "fsck $_dev: errors corrected" ;;
	*) log "WARNING: fsck $_dev exited $_rc (continuing boot)" ;;
	esac
	return 0
}

fsck_before_mount() {
	# Kernel already mounted / — remount RO so e2fsck/fsck will accept it.
	_rootdev=
	if is_mounted /; then
		log "remount,ro / for fsck"
		mount -o remount,ro / 2>/dev/null || true
		_rootdev=$(awk '$2 == "/" { print $1; exit }' /proc/mounts 2>/dev/null || true)
	fi

	if [ -r /etc/fstab ]; then
		if command -v fsck >/dev/null 2>&1; then
			_rc=0
			log "fsck -A -y (fstab pass numbers)"
			fsck -A -y -T || _rc=$?
			case $_rc in
			0) ;;
			1) log "fsck -A: errors corrected" ;;
			*) log "WARNING: fsck -A exited $_rc (continuing boot)" ;;
			esac
		fi
		while read -r _fsck_dev _fsck_mp _fsck_type _fsck_opts _fsck_dump _fsck_pass _rest; do
			case "$_fsck_dev" in
			'' | \#*) continue ;;
			esac
			case "$_fsck_type" in
			proc | sysfs | devtmpfs | devpts | tmpfs | ramfs | cgroup* | \
			overlay | squashfs | nfs* | cifs | autofs | debugfs | \
			securityfs | pstore | bpf | tracefs | hugetlbfs | mqueue | \
			configfs | fusectl | swap)
				continue
				;;
			esac
			fsck_one "$_fsck_dev"
		done < /etc/fstab
	fi

	# Explicit root check: fsck_one skips mounted devices, and BusyBox
	# fsck -A may be a stub. Root is RO now, so force-repair is safe.
	if [ -n "${_rootdev:-}" ]; then
		_rc=0
		if command -v fsck >/dev/null 2>&1; then
			log "fsck -y $_rootdev (root, remounted ro)"
			fsck -y -T "$_rootdev" || _rc=$?
		elif command -v e2fsck >/dev/null 2>&1; then
			log "e2fsck -y $_rootdev (root, remounted ro)"
			e2fsck -y "$_rootdev" || _rc=$?
		else
			return 0
		fi
		case $_rc in
		0) ;;
		1) log "fsck $_rootdev: errors corrected" ;;
		*) log "WARNING: fsck $_rootdev exited $_rc (continuing boot)" ;;
		esac
	fi
}

# fsck hub /data candidates (not in fstab) before any probe/mount.
fsck_data_candidates() {
	for _dev in /dev/mmcblk0p3 /dev/disk/by-label/bigfred-data; do
		fsck_one "$_dev"
	done
	if command -v findfs >/dev/null 2>&1; then
		_lab=$(findfs LABEL=bigfred-data 2>/dev/null || true)
		[ -n "${_lab:-}" ] && fsck_one "$_lab"
	fi
	# Any NVMe partition that already has a filesystem (migrated or raw).
	for _d in /dev/nvme*n*; do
		[ -e "$_d" ] || continue
		_base=$(basename "$_d")
		for _p in /dev/${_base}p*; do
			[ -b "$_p" ] || continue
			fsck_one "$_p"
		done
	done
}

# --- essential pseudo filesystems (needed before fstab / fsck) ---
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

# --- fsck real filesystems before any block mount ---
fsck_before_mount
fsck_data_candidates

# --- fstab (root + tmpfs; /data is mounted below) ---
if [ -r /etc/fstab ]; then
	log "mount -a"
	mount -a || true
else
	log "no /etc/fstab; skipping mount -a"
fi

mkdir -p /var/log /var/run /run "$DATA_ROOT"

# --- /data partition (hub default root only) ---
# Prefer NVMe p1 if it was already migrated (ext4 + marker file); otherwise
# fall back to the microSD data partition. prepare-nvme creates the marker
# after a successful copy, so subsequent boots mount NVMe directly without
# going through the SD.
NVME_MARKER=".bigfred-nvme"

nvme_data_part() {
	# Echoes the first NVMe partition that is ext4 and carries the migration
	# marker, or empty if none.
	for _d in /dev/nvme*n*; do
		[ -e "$_d" ] || continue
		_base=$(basename "$_d")
		for _p in /dev/${_base}p*; do
			[ -b "$_p" ] || continue
			_fstype=$(blkid -o value -s TYPE "$_p" 2>/dev/null || true)
			[ "$_fstype" = "ext4" ] || continue
			# Probe marker without leaving the mount in place.
			_tmp=$(mktemp -d "${TMPDIR:-/tmp}/nvme-probe.XXXXXX")
			if mount -t ext4 -o ro "$_p" "$_tmp" 2>/dev/null; then
				_ok=0
				[ -e "$_tmp/$NVME_MARKER" ] && _ok=1
				umount "$_tmp" 2>/dev/null || umount -l "$_tmp" 2>/dev/null || true
				rmdir "$_tmp" 2>/dev/null || true
				[ "$_ok" = "1" ] && echo "$_p" && return 0
			else
				rmdir "$_tmp" 2>/dev/null || true
			fi
		done
	done
	return 1
}

mount_data() {
	if is_mounted /data; then
		return 0
	fi
	# Shared mount options: noatime + commit=15 (batch journal flushes;
	# 15s avoids stacking with SQLite/Redis write windows on hard power loss).
	_data_opts="rw,noatime,commit=15"
	# 1) NVMe p1 (ext4 + marker) — migrated data lives here.
	_nvme=$(nvme_data_part || true)
	if [ -n "$_nvme" ] && [ -b "$_nvme" ]; then
		log "mounting NVMe data partition: $_nvme"
		mount -t ext4 -o "$_data_opts" "$_nvme" /data && return 0
		log "WARNING: NVMe marker present but mount failed — trying SD"
	fi
	# 2) microSD data partition by label, then by device.
	if command -v findfs >/dev/null 2>&1; then
		_dev=$(findfs LABEL=bigfred-data 2>/dev/null || true)
		if [ -n "${_dev:-}" ] && [ -b "$_dev" ]; then
			mount -t ext4 -o "$_data_opts" "$_dev" /data && return 0
		fi
	fi
	for _dev in /dev/disk/by-label/bigfred-data /dev/mmcblk0p3; do
		if [ -b "$_dev" ] || [ -e "$_dev" ]; then
			if mount -t ext4 -o "$_data_opts" "$_dev" /data 2>/dev/null; then
				return 0
			fi
		fi
	done
	# 3) Unformatted microSD data partition — create once (empty TYPE only).
	for _dev in /dev/mmcblk0p3; do
		if [ -b "$_dev" ]; then
			_fstype=$(blkid -o value -s TYPE "$_dev" 2>/dev/null || true)
			if [ -z "$_fstype" ]; then
				log "formatting $_dev as ext4 (LABEL=bigfred-data)"
				mkfs.ext4 -F -L bigfred-data "$_dev"
				mount -t ext4 -o "$_data_opts" "$_dev" /data && return 0
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
	# If /data is on the microSD and NVMe is present but not yet migrated,
	# prepare-nvme will create/format the NVMe partition, copy /data onto
	# it, and switch the mount. On later boots mount_data() above picks
	# the NVMe directly and prepare-nvme is a no-op.
	if [ -x /usr/sbin/prepare-nvme ]; then
		log "prepare-nvme"
		_rc=0
		/usr/sbin/prepare-nvme || _rc=$?
		if [ "$_rc" -ne 0 ]; then
			log "WARNING: prepare-nvme failed (exit $_rc) — keeping current /data"
		elif is_mounted /data; then
			# prepare-nvme remounts /data with rw,noatime only; re-apply
			# commit=15 so the ext4 journal window stays aligned with the
			# vm.dirty_* sysctl tuning for the rest of this boot.
			log "re-applying commit=15 on /data after NVMe migration"
			mount -o remount,rw,noatime,commit=15 /data 2>/dev/null \
				|| log "WARNING: could not remount /data with commit=15"
		fi
	fi
else
	log "using custom data root $DATA_ROOT (skip partition mount)"
	mkdir -p "$DATA_ROOT"
fi

# --- directories ---
mkdir -p "$DATA_ROOT/etc" "$DATA_ROOT/logs" "$DATA_ROOT/run"
mkdir -p "$DATA_ROOT/var/db/redis" "$DATA_ROOT/var/db/bigfred"
mkdir -p "$DATA_ROOT/var/lib/alloy" \
	"$DATA_ROOT/var/lib/victoriametrics" \
	"$DATA_ROOT/var/lib/grafana/data" \
	"$DATA_ROOT/var/lib/grafana/plugins"
mkdir -p "$DATA_ROOT/opt/bigfred/bin"
# Service logs live on tmpfs under /data/logs (RAM; lost on reboot) to
# avoid SD wear. Mount after /data so the mountpoint exists on the RW
# partition. Cap at 64m; rotate-hub-logs enforces size/retention.
if [ "$DATA_ROOT" = /data ] && is_mounted /data && ! is_mounted /data/logs; then
	log "mounting tmpfs /data/logs (64m)"
	if ! mount -t tmpfs -o mode=0755,size=64m tmpfs /data/logs 2>/tmp/tmpfs-logs.err; then
		# Do NOT swallow this silently: a failed tmpfs mount means service
		# logs would land on the ext4 /data partition and silently defeat
		# the SD-wear reduction goal. Surface the error so the operator
		# can fix it, but keep booting (logs are non-fatal).
		log "WARNING: tmpfs /data/logs mount failed — logs will be written"
		log "WARNING: to persistent /data/logs (SD wear!). err: $(cat /tmp/tmpfs-logs.err 2>/dev/null)"
	fi
	rm -f /tmp/tmpfs-logs.err
fi
mkdir -p "$DATA_ROOT/logs/bigfred" "$DATA_ROOT/logs/redis" \
	"$DATA_ROOT/logs/alloy" "$DATA_ROOT/logs/grafana"
mkdir -p "$DATA_ROOT/etc/microinit" \
	"$DATA_ROOT/etc/microinit.d/services/infra" \
	"$DATA_ROOT/etc/microinit.d/services/dcc-bus" \
	"$DATA_ROOT/etc/microinit.d/services/os"

# Service $HOME on RO root: tmpfs so accidental writes to HOME do not fail
# and do not pollute /data. No login shell (users.table: /bin/false).
mkdir -p /home/bigfred
if ! is_mounted /home/bigfred; then
	mount -t tmpfs -o mode=0750,uid=0,gid=0,size=8m tmpfs /home/bigfred 2>/dev/null || true
fi
if id bigfred >/dev/null 2>&1 && is_mounted /home/bigfred; then
	chown bigfred:bigfred /home/bigfred 2>/dev/null || true
	chmod 0750 /home/bigfred 2>/dev/null || true
fi

# --- ownership for non-root services (best-effort if users exist) ---
chown_if() {
	_user=$1
	shift
	if id "$_user" >/dev/null 2>&1; then
		chown -R "$_user:$_user" "$@" 2>/dev/null || true
	fi
}
chown_if redis "$DATA_ROOT/var/db/redis" "$DATA_ROOT/logs/redis"
chown_if alloy "$DATA_ROOT/var/lib/alloy" "$DATA_ROOT/logs/alloy"
chown_if metrics \
	"$DATA_ROOT/var/lib/victoriametrics" \
	"$DATA_ROOT/var/lib/grafana" \
	"$DATA_ROOT/logs/grafana"
# SQLite lives under var/db/bigfred/; chown the directory so WAL/SHM inherit.
# Only loco-server-managed drop-in groups are bigfred-owned; OS services stay root.
# Parents microinit.d/ and services/ stay root-owned but group bigfred so the
# loco-server can traverse (r-x) without being able to create/delete sibling
# groups such as os/ (no group write on parents).
chown_if bigfred \
	"$DATA_ROOT/var/db/bigfred" \
	"$DATA_ROOT/logs/bigfred" \
	"$DATA_ROOT/etc/microinit.d/services/infra" \
	"$DATA_ROOT/etc/microinit.d/services/dcc-bus"
if id bigfred >/dev/null 2>&1; then
	chown root:bigfred \
		"$DATA_ROOT/etc/microinit.d" \
		"$DATA_ROOT/etc/microinit.d/services" 2>/dev/null || true
fi
chmod 0750 "$DATA_ROOT/var/db/redis" "$DATA_ROOT/var/db/bigfred" \
	"$DATA_ROOT/var/lib/alloy" "$DATA_ROOT/var/lib/victoriametrics" 2>/dev/null || true
chmod 0750 "$DATA_ROOT/logs/redis" "$DATA_ROOT/logs/alloy" "$DATA_ROOT/logs/bigfred" "$DATA_ROOT/logs/grafana" 2>/dev/null || true
chmod 0750 "$DATA_ROOT/etc/microinit.d" "$DATA_ROOT/etc/microinit.d/services" 2>/dev/null || true
chmod 0750 "$DATA_ROOT/etc/microinit.d/services/infra" \
	"$DATA_ROOT/etc/microinit.d/services/dcc-bus" 2>/dev/null || true
chmod 0755 "$DATA_ROOT/etc/microinit.d/services/os" 2>/dev/null || true
if [ -f "$DATA_ROOT/etc/redis.conf" ]; then
	chmod 0640 "$DATA_ROOT/etc/redis.conf"
	if id redis >/dev/null 2>&1; then
		chown root:redis "$DATA_ROOT/etc/redis.conf" 2>/dev/null || true
	fi
fi
if [ -f "$DATA_ROOT/var/db/bigfred/bigfred.sqlite3" ] && id bigfred >/dev/null 2>&1; then
	chown bigfred:bigfred "$DATA_ROOT/var/db/bigfred"/bigfred.sqlite3* 2>/dev/null || true
	chmod 0640 "$DATA_ROOT/var/db/bigfred"/bigfred.sqlite3* 2>/dev/null || true
fi

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
# microdns config is regenerated every boot (operator edits not preserved by design).
if [ -e /etc/microdns/microdns.json ]; then
	mkdir -p "$DATA_ROOT/etc"
	cp /etc/microdns/microdns.json "$DATA_ROOT/etc/microdns.json"
	chmod 644 "$DATA_ROOT/etc/microdns.json"
fi
# OS-owned microinit services live as one JSON file per service under
# microinit.d/services/os/. Always refresh from the image so upgrades pick up
# new services (e.g. microdns) even when /data/etc/microinit.json already exists.
# Drop-ins with the same name override any leftover entries in the main file.
OS_DROPINS_SRC=/etc/microinit.d/services/os
OS_DROPINS_DST="$DATA_ROOT/etc/microinit.d/services/os"
mkdir -p "$OS_DROPINS_DST"
if [ -d "$OS_DROPINS_SRC" ]; then
	for f in "$OS_DROPINS_DST"/*.json; do
		[ -e "$f" ] || continue
		base=$(basename "$f")
		[ -f "$OS_DROPINS_SRC/$base" ] || rm -f "$f"
	done
	for f in "$OS_DROPINS_SRC"/*.json; do
		[ -e "$f" ] || continue
		cp "$f" "$OS_DROPINS_DST/"
		chmod 644 "$OS_DROPINS_DST/$(basename "$f")"
	done
fi
# Main microinit.json keeps globals only (services[] empty in the image template).
# Seed only if missing so operator customizations of logs/socket/otel remain.
# microinit.json stays root-owned (PID 1 reads/writes it); loco-server drop-ins
# under infra/ and dcc-bus/ are chowned to bigfred below.
seed /etc/microinit/microinit.json "$DATA_ROOT/etc/microinit.json" 644
seed /etc/microinit/microinit.json "$DATA_ROOT/etc/microinit.json.example" 644
seed /etc/microinit/otel.env.example "$DATA_ROOT/etc/otel.env.example" 644

if [ -f "$DATA_ROOT/etc/redis.conf" ]; then
	chmod 0640 "$DATA_ROOT/etc/redis.conf"
	if id redis >/dev/null 2>&1; then
		chown root:redis "$DATA_ROOT/etc/redis.conf" 2>/dev/null || true
	fi
fi
if id bigfred >/dev/null 2>&1; then
	# Re-assert after OS drop-in refresh: parents stay root:bigfred (traverse),
	# loco-server groups stay bigfred-owned.
	chown root:bigfred \
		"$DATA_ROOT/etc/microinit.d" \
		"$DATA_ROOT/etc/microinit.d/services" 2>/dev/null || true
	chown -R bigfred:bigfred \
		"$DATA_ROOT/etc/microinit.d/services/infra" \
		"$DATA_ROOT/etc/microinit.d/services/dcc-bus" 2>/dev/null || true
fi
# Image-managed OS drop-ins must stay root-owned.
if [ -d "$OS_DROPINS_DST" ]; then
	chown -R root:root "$OS_DROPINS_DST" 2>/dev/null || true
fi

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

# --- persistent root SSH keys via bind-mounted .ssh ---
mkdir -p "$DATA_ROOT/root/.ssh"
chmod 700 "$DATA_ROOT/root/.ssh"
if [ ! -d /root/.ssh ]; then
	# Root is RO after fsck; remount RW only long enough to create the mount point
	# (needed on images built before post-build seeded /root/.ssh).
	mount -o remount,rw / 2>/dev/null || true
	mkdir -p /root/.ssh
	chmod 700 /root/.ssh
fi
if [ "$DATA_ROOT" = /data ]; then
	if is_mounted /data && ! is_mounted /root/.ssh; then
		mount --bind "$DATA_ROOT/root/.ssh" /root/.ssh 2>/dev/null || true
	fi
elif ! is_mounted /root/.ssh; then
	mount --bind "$DATA_ROOT/root/.ssh" /root/.ssh 2>/dev/null || true
fi

# --- timezone: persist operator choice from /data/etc (bind over RO /etc/localtime) ---
# Same category as shadow: must be ready before any supervised process starts
# (microinit launches dependsOn:[] services in parallel).
if [ -r "$DATA_ROOT/etc/timezone" ]; then
	_tz=$(tr -d '[:space:]' < "$DATA_ROOT/etc/timezone" 2>/dev/null || true)
	if [ -n "$_tz" ] && [ -e "/usr/share/zoneinfo/$_tz" ]; then
		cp -f "/usr/share/zoneinfo/$_tz" "$DATA_ROOT/etc/localtime"
	fi
fi
if [ -f "$DATA_ROOT/etc/localtime" ] && ! is_mounted /etc/localtime; then
	mount --bind "$DATA_ROOT/etc/localtime" /etc/localtime 2>/dev/null || true
fi

# --- fake-hwclock: restore last known wall clock (no RTC on hub) ---
if [ -r "$DATA_ROOT/etc/fake-hwclock" ] && command -v date >/dev/null 2>&1; then
	_epoch=$(tr -dc '0-9' < "$DATA_ROOT/etc/fake-hwclock" 2>/dev/null || true)
	_now=$(date -u +%s 2>/dev/null || echo 0)
	if [ -n "$_epoch" ] && [ "$_now" -lt "$_epoch" ] 2>/dev/null; then
		log "fake-hwclock: restore $_epoch"
		date -u -s "@$_epoch" 2>/dev/null || true
	fi
fi

# Enforce read-only root (hub policy; ignore failure on RW systems)
mount -o remount,ro / 2>/dev/null || true

log "done (DATA_ROOT=$DATA_ROOT)"
exit 0
