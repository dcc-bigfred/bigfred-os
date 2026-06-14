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
mkdir -p "${BOOT_DIR}"

cp -v "${BINARIES_DIR}/Image" "${BOOT_DIR}/"
for dtb in "${BINARIES_DIR}"/*.dtb; do
	[ -f "$dtb" ] && cp -v "$dtb" "${BOOT_DIR}/"
done

RPI_FW="$(ls -d "${BUILD_DIR}"/build/rpi-firmware-* 2>/dev/null | head -1)"
if [ -n "${RPI_FW}" ] && [ -d "${RPI_FW}" ]; then
	cp -v "${RPI_FW}/bootcode.bin" "${BOOT_DIR}/" 2>/dev/null || true
	cp -v "${RPI_FW}"/start*.elf "${RPI_FW}"/fixup*.dat "${BOOT_DIR}/" 2>/dev/null || true
fi

# rpi-firmware installs these under BINARIES_DIR (not target/boot/).
RPI_FW_IMG="${BINARIES_DIR}/rpi-firmware"
if [ -f "${RPI_FW_IMG}/config.txt" ]; then
	cp -v "${RPI_FW_IMG}/config.txt" "${BOOT_DIR}/config.txt"
else
	cp -v "${BOARD_DIR}/config.txt" "${BOOT_DIR}/config.txt"
fi
if [ -f "${RPI_FW_IMG}/cmdline.txt" ]; then
	cp -v "${RPI_FW_IMG}/cmdline.txt" "${BOOT_DIR}/cmdline.txt"
else
	cp -v "${BOARD_DIR}/cmdline.txt" "${BOOT_DIR}/cmdline.txt"
fi

# mtools image for genimage
BOOT_MBR="${BINARIES_DIR}/boot.vfat"
rm -f "${BOOT_MBR}"
# -C size is in 1024-byte blocks (65536 × 1 KiB = 64 MiB)
"${HOST_DIR}/sbin/mkdosfs" -n BOOT -C "${BOOT_MBR}" $((64 * 1024))
for f in "${BOOT_DIR}"/*; do
	[ -e "$f" ] || continue
	"${HOST_DIR}/bin/mcopy" -i "${BOOT_MBR}" "$f" ::/
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
