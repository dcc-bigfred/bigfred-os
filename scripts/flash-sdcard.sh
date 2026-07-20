#!/usr/bin/env bash
# Interactively flash a built hub OS image onto an SD card (mmcblk* only).
#
# Usage:
#   sudo ./scripts/flash-sdcard.sh [path/to/hub-nvme.img]
#
# Default image: os/output/images/hub-nvme.img (or sdcard.img symlink).
# Build first: make image

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEFAULT_IMG="${ROOT}/os/output/images/hub-nvme.img"
ALT_IMG="${ROOT}/os/output/images/sdcard.img"

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
	cat <<EOF
Usage: sudo $0 [image.img]

Find removable mmcblk* block devices, let you pick one, then flash the hub OS
image built under os/output/images/.

Only /dev/mmcblk<N> whole disks are allowed (not partitions, not sdX/nvme).
You will be asked to confirm before writing.

Build the image first:
  make image
EOF
	exit 0
fi

if [ "$(id -u)" -ne 0 ]; then
	echo "error: run as root (e.g. sudo $0)" >&2
	exit 1
fi

resolve_image() {
	local candidate="$1"
	if [ -n "$candidate" ]; then
		if [ -f "$candidate" ]; then
			printf '%s\n' "$(readlink -f "$candidate")"
			return 0
		fi
		echo "error: image not found: $candidate" >&2
		return 1
	fi
	if [ -f "$DEFAULT_IMG" ]; then
		printf '%s\n' "$(readlink -f "$DEFAULT_IMG")"
		return 0
	fi
	if [ -f "$ALT_IMG" ]; then
		printf '%s\n' "$(readlink -f "$ALT_IMG")"
		return 0
	fi
	echo "error: no image found (build with: make image)" >&2
	echo "       expected: $DEFAULT_IMG" >&2
	return 1
}

# Whole-disk mmcblk only — reject mmcblk0p1, sd*, nvme*, etc.
is_allowed_mmc_disk() {
	local dev="$1"
	[[ "$dev" =~ ^/dev/mmcblk[0-9]+$ ]]
}

discover_mmc_disks() {
	local sys name dev
	for sys in /sys/block/mmcblk*; do
		[ -d "$sys" ] || continue
		name="$(basename "$sys")"
		dev="/dev/${name}"
		[ -b "$dev" ] || continue
		is_allowed_mmc_disk "$dev" || continue
		printf '%s\n' "$dev"
	done
}

describe_disk() {
	local dev="$1"
	if command -v lsblk >/dev/null 2>&1; then
		lsblk -d -n -o SIZE,MODEL,TRAN "$dev" 2>/dev/null | tr '\n' ' '
	else
		local size
		size="$(cat "/sys/block/$(basename "$dev")/size" 2>/dev/null || echo 0)"
		echo "$(( size * 512 )) bytes"
	fi
}

umount_disk_partitions() {
	local dev="$1" p
	for p in "${dev}"p*; do
		[ -b "$p" ] || continue
		if findmnt -n "$p" >/dev/null 2>&1; then
			echo "Unmounting $p ..."
			umount "$p"
		fi
	done
}

IMG="$(resolve_image "${1:-}")"

mapfile -t DISKS < <(discover_mmc_disks | sort -V)
if [ "${#DISKS[@]}" -eq 0 ]; then
	echo "No mmcblk* disks found." >&2
	echo "Insert an SD card (built-in reader) and retry." >&2
	exit 1
fi

echo "Hub OS image: $IMG"
echo "Size: $(du -h "$IMG" | awk '{print $1}')"
echo ""
echo "Available mmcblk devices:"
echo ""

SELECTED_DEV=""
if [ "${#DISKS[@]}" -eq 1 ]; then
	SELECTED_DEV="${DISKS[0]}"
	echo "  [1] ${SELECTED_DEV}  $(describe_disk "$SELECTED_DEV")"
	echo ""
	printf "Use %s? [y/N]: " "$SELECTED_DEV"
	read -r pick_one
	case "$pick_one" in
	y|Y|yes|YES) ;;
	*) echo "Aborted."; exit 1 ;;
	esac
else
	i=1
	for dev in "${DISKS[@]}"; do
		printf "  [%d] %s  %s\n" "$i" "$dev" "$(describe_disk "$dev")"
		i=$((i + 1))
	done
	echo ""
	while true; do
		printf "Select device [1-%d] (or q to quit): " "${#DISKS[@]}"
		read -r choice
		case "$choice" in
		q|Q)
			echo "Aborted."
			exit 1
			;;
		esac
		if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "${#DISKS[@]}" ]; then
			SELECTED_DEV="${DISKS[$((choice - 1))]}"
			break
		fi
		echo "Invalid choice."
	done
fi

if ! is_allowed_mmc_disk "$SELECTED_DEV"; then
	echo "error: refused device (not an allowed mmcblk disk): $SELECTED_DEV" >&2
	exit 1
fi

echo ""
echo "WARNING: all data on ${SELECTED_DEV} will be destroyed."
echo "Image:  ${IMG}"
echo "Target: ${SELECTED_DEV}  $(describe_disk "$SELECTED_DEV")"
printf "Type YES to flash: "
read -r confirm
if [ "$confirm" != YES ]; then
	echo "Aborted."
	exit 1
fi

if ! is_allowed_mmc_disk "$SELECTED_DEV"; then
	echo "error: refused device at flash time: $SELECTED_DEV" >&2
	exit 1
fi

umount_disk_partitions "$SELECTED_DEV"

echo "Writing image ..."
dd if="$IMG" of="$SELECTED_DEV" bs=4M conv=fsync status=progress
sync
partprobe "$SELECTED_DEV" 2>/dev/null || true

echo "Done. Insert the SD card into the Raspberry Pi and boot."
