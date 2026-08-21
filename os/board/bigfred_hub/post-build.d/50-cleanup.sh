# Skip leftover biginit binary if present in apps/.bin from older trees
rm -f "${TARGET_DIR}/usr/sbin/biginit"
# micronet owns dnsmasq; BR2_INIT_NONE should not install S80dnsmasq.
rm -f "${TARGET_DIR}/etc/init.d/S80dnsmasq"
