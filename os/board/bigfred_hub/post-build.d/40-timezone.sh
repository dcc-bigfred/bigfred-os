# Default timezone (Europe/Warsaw); operator override lives on /data via early-boot bind.
if [ -e "${TARGET_DIR}/usr/share/zoneinfo/Europe/Warsaw" ]; then
	ln -sfn /usr/share/zoneinfo/Europe/Warsaw "${TARGET_DIR}/etc/localtime"
	printf 'Europe/Warsaw\n' > "${TARGET_DIR}/etc/timezone"
fi
