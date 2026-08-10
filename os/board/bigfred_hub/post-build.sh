#!/bin/sh
# Post-build runner: executes post-build.d/*.sh in lexical order.
# Buildroot invokes this as BR2_ROOTFS_POST_BUILD_SCRIPT (TARGET_DIR set).

set -e

HUB="${BR2_EXTERNAL_BIGFRED_HUB_PATH:-$(dirname "$0")/../..}"
REPO_ROOT="$(cd "${HUB}/.." && pwd)"

# Shared commit identity (consumed by post-build.d/60-identity.sh).
COMMIT="unknown"
if command -v git >/dev/null 2>&1 && [ -d "${REPO_ROOT}/.git" ]; then
	COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || true)"
	[ -n "${COMMIT}" ] || COMMIT="unknown"
elif [ -n "${GITHUB_SHA:-}" ]; then
	COMMIT="${GITHUB_SHA}"
fi
export HUB REPO_ROOT COMMIT

D="${HUB}/board/bigfred_hub/post-build.d"
for script in "${D}"/*.sh; do
	[ -f "${script}" ] || continue
	echo "post-build: $(basename "${script}")"
	# shellcheck source=/dev/null
	. "${script}"
done

# WiFi firmware for Raspberry Pi 5 (CYW43455 SDIO, same part as Pi 4).
# linux-firmware's BRCM_BCM43XXX ships NVRAM for Pi 3B+ and Pi 4 but not Pi 5;
# CYPRESS_CYW43XXX ships the .bin/.clm_blob under cypress/ (brcmfmac falls back
# to that path). Symlink the Pi 5 board NVRAM to the Pi 4 NVRAM and expose the
# Cypress .bin under brcm/ so the primary firmware path also resolves.
#
# The NVRAM substitution is a stopgap: it carries the Pi 4 board calibration
# (antenna trim, regulatory limits), which is close but not authored for the Pi
# 5 layout. Replace it with the real Pi 5 NVRAM once linux-firmware carries it;
# until then check `dmesg | grep brcmfmac` on a booted board for firmware/NVRAM
# load errors after changing anything here.
FW_BRCM="${TARGET_DIR}/lib/firmware/brcm"
FW_CYP="${TARGET_DIR}/lib/firmware/cypress"
# brcm/ may not exist if only the Cypress sub-option installed files; the
# symlinks below are what make the primary brcm/ path resolve at all.
mkdir -p "${FW_BRCM}"
if [ -d "${FW_BRCM}" ]; then
	if [ -f "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,4-model-b.txt" ] && \
	   [ ! -e "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.txt" ]; then
		ln -sf "brcmfmac43455-sdio.raspberrypi,4-model-b.txt" \
			"${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.txt"
	fi
	if [ -f "${FW_CYP}/cyfmac43455-sdio.bin" ] && \
	   [ ! -e "${FW_BRCM}/brcmfmac43455-sdio.bin" ]; then
		ln -sf "../cypress/cyfmac43455-sdio.bin" \
			"${FW_BRCM}/brcmfmac43455-sdio.bin"
	fi
	if [ -f "${FW_CYP}/cyfmac43455-sdio.clm_blob" ] && \
	   [ ! -e "${FW_BRCM}/brcmfmac43455-sdio.clm_blob" ]; then
		ln -sf "../cypress/cyfmac43455-sdio.clm_blob" \
			"${FW_BRCM}/brcmfmac43455-sdio.clm_blob"
	fi
fi

exit 0
