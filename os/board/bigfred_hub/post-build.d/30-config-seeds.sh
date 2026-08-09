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
