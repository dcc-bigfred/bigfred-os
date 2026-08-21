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

# Overlay copies preserve git file mode. microinit exec's /etc/init.d/* as
# cmd; a 0644 script fails with "permission denied" (bigfred-wizard hit this).
if [ -d "${TARGET_DIR}/etc/init.d" ]; then
	find "${TARGET_DIR}/etc/init.d" -type f -exec chmod 0755 {} +
fi

# Pi 5 brcmfmac looks up board-specific names first
# (`brcmfmac43455-sdio.raspberrypi,5-model-b.{bin,txt,clm_blob}`). The
# brcmfmac_sdio-firmware-rpi package ships those; keep fallbacks if a file
# is missing so a stale package still enumerates wlan0.
FW_BRCM="${TARGET_DIR}/lib/firmware/brcm"
FW_CYP="${TARGET_DIR}/lib/firmware/cypress"
mkdir -p "${FW_BRCM}"
link_fw() {
	_dst=$1
	_src=$2
	if [ -e "${_src}" ] && [ ! -e "${_dst}" ]; then
		ln -sf "$(basename "${_src}")" "${_dst}"
	fi
}
if [ -d "${FW_BRCM}" ]; then
	link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.txt" \
		"${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,4-model-b.txt"
	link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.txt" \
		"${FW_BRCM}/brcmfmac43455-sdio.txt"
	link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.bin" \
		"${FW_BRCM}/brcmfmac43455-sdio.bin"
	link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.clm_blob" \
		"${FW_BRCM}/brcmfmac43455-sdio.clm_blob"
	if [ -f "${FW_CYP}/cyfmac43455-sdio.bin" ] && \
	   [ ! -e "${FW_BRCM}/brcmfmac43455-sdio.bin" ]; then
		ln -sf "../cypress/cyfmac43455-sdio.bin" \
			"${FW_BRCM}/brcmfmac43455-sdio.bin"
		link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.bin" \
			"${FW_BRCM}/brcmfmac43455-sdio.bin"
	fi
	if [ -f "${FW_CYP}/cyfmac43455-sdio.clm_blob" ] && \
	   [ ! -e "${FW_BRCM}/brcmfmac43455-sdio.clm_blob" ]; then
		ln -sf "../cypress/cyfmac43455-sdio.clm_blob" \
			"${FW_BRCM}/brcmfmac43455-sdio.clm_blob"
		link_fw "${FW_BRCM}/brcmfmac43455-sdio.raspberrypi,5-model-b.clm_blob" \
			"${FW_BRCM}/brcmfmac43455-sdio.clm_blob"
	fi
fi

exit 0
