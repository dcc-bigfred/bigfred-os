# Install hub app binaries (make -C apps build → apps/.bin/)
APPS_BIN="${HUB}/../apps/.bin"
if [ -d "${APPS_BIN}" ]; then
	installed=0
	for bin in "${APPS_BIN}"/*; do
		[ -f "$bin" ] || continue
		case "$(basename "$bin")" in
			bigfred-os-ui-*)
				# Installed below as /usr/sbin/bigfred-os-ui
				continue
				;;
			biginit)
				# Replaced by microinit (package/microinit)
				continue
				;;
			configure-ethernet|configure-dhcp)
				# Replaced by micronet (package/micronet)
				continue
				;;
		esac
		[ -x "$bin" ] || chmod 755 "$bin"
		name=$(basename "$bin")
		install -D -m 0755 "$bin" "${TARGET_DIR}/usr/sbin/${name}"
		installed=$((installed + 1))
	done
	UI_TARGET="${APPS_BIN}/bigfred-os-ui-linux-arm64"
	if [ -f "${UI_TARGET}" ]; then
		install -D -m 0755 "${UI_TARGET}" "${TARGET_DIR}/usr/sbin/bigfred-os-ui"
		installed=$((installed + 1))
	fi
	if [ "$installed" -eq 0 ]; then
		echo "warning: ${APPS_BIN} is empty — run: make -C apps build" >&2
	fi
else
	echo "warning: ${APPS_BIN} missing — run: make -C apps build" >&2
fi
