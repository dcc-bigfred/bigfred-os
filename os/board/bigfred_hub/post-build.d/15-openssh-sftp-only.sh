# Dropbear listens on :22; keep only OpenSSH's sftp-server for `scp` / SFTP clients.
if [ -f "${TARGET_DIR}/usr/libexec/sftp-server" ]; then
	rm -f "${TARGET_DIR}/usr/sbin/sshd"
	rm -f "${TARGET_DIR}/usr/libexec/sshd-session"
	rm -f "${TARGET_DIR}/etc/init.d/S50sshd"
fi
