#!/bin/sh
# Assemble boot + root + /data into output/images/hub-nvme.img

set -e

BOARD_DIR="$(cd "$(dirname "$0")" && pwd)"
HUB_DIR="$(cd "${BOARD_DIR}/../.." && pwd)"
GENIMAGE_CFG="${BOARD_DIR}/genimage.cfg"
BINARIES_DIR="${BINARIES_DIR:?BINARIES_DIR not set}"
BUILD_DIR="${BUILD_DIR:?BUILD_DIR not set}"
TARGET_DIR="${TARGET_DIR:?TARGET_DIR not set}"

BOOT_DIR="${BINARIES_DIR}/boot"
ROOTFS="${BINARIES_DIR}/rootfs.ext2"
DATA_IMG="${BINARIES_DIR}/data.ext2"
OUTPUT_IMG="${BINARIES_DIR}/hub-nvme.img"

# --- boot partition contents (FAT) ---
rm -rf "${BOOT_DIR}"
mkdir -p "${BOOT_DIR}/overlays"

cp -v "${BINARIES_DIR}/Image" "${BOOT_DIR}/"
for dtb in "${BINARIES_DIR}"/*.dtb; do
	[ -f "$dtb" ] && cp -v "$dtb" "${BOOT_DIR}/"
done

# Boot config from board (source of truth; includes D0 device_tree=).
cp -v "${BOARD_DIR}/config.txt" "${BOOT_DIR}/config.txt"
cp -v "${BOARD_DIR}/cmdline.txt" "${BOOT_DIR}/cmdline.txt"

# Optional: firmware DT overlays (bcm2712d0.dtbo etc.) from rpi-firmware package.
RPI_FW_IMG="${BINARIES_DIR}/rpi-firmware"
if [ -d "${RPI_FW_IMG}/overlays" ]; then
	cp -a "${RPI_FW_IMG}/overlays/." "${BOOT_DIR}/overlays/"
fi
# Ensure bcm2712d0 overlay from the kernel tree is present even without firmware overlays.
KERNEL_OVLAY="$(ls -d "${BUILD_DIR}"/build/linux-custom/arch/arm/boot/dts/overlays 2>/dev/null | head -1)"
if [ -f "${KERNEL_OVLAY}/bcm2712d0.dtbo" ]; then
	cp -v "${KERNEL_OVLAY}/bcm2712d0.dtbo" "${BOOT_DIR}/overlays/"
fi

# mtools image for genimage
BOOT_MBR="${BINARIES_DIR}/boot.vfat"
rm -f "${BOOT_MBR}"
# -C size is in 1024-byte blocks (65536 × 1 KiB = 64 MiB)
"${HOST_DIR}/sbin/mkdosfs" -n BOOT -C "${BOOT_MBR}" $((64 * 1024))
for f in "${BOOT_DIR}"/*; do
	[ -e "$f" ] || continue
	if [ -d "$f" ]; then
		"${HOST_DIR}/bin/mmd" -i "${BOOT_MBR}" "::/$(basename "$f")" 2>/dev/null || true
		for sf in "$f"/*; do
			[ -f "$sf" ] || continue
			"${HOST_DIR}/bin/mcopy" -i "${BOOT_MBR}" "$sf" "::/$(basename "$f")/"
		done
	else
		"${HOST_DIR}/bin/mcopy" -i "${BOOT_MBR}" "$f" ::/
	fi
done

# --- empty /data ext4 ---
rm -f "${DATA_IMG}"
"${HOST_DIR}/sbin/mke2fs" -t ext4 -L bigfred-data -d "${TARGET_DIR}/data" \
	"${DATA_IMG}" 512M

export BOOTIMAGE="${BOOT_MBR}"
export ROOTFSIMAGE="${ROOTFS}"
export DATAIMAGE="${DATA_IMG}"

rm -f "${OUTPUT_IMG}"
"${HOST_DIR}/bin/genimage" \
	--rootpath "${TARGET_DIR}" \
	--tmppath "${BUILD_DIR}/genimage.tmp" \
	--inputpath "${BINARIES_DIR}" \
	--outputpath "${BINARIES_DIR}" \
	--config "${GENIMAGE_CFG}"

# Symlink for convenience (doc §8.10 sdcard.img naming)
ln -sf hub-nvme.img "${BINARIES_DIR}/sdcard.img"

echo "Hub image: ${OUTPUT_IMG}"
