# SPDX-License-Identifier: MIT
# Prebuilt Raspberry Pi 5 kernel from dcc-bigfred/hub-kernel Releases.
# Pin VERSION + linux-prebuilt.hash (Grafana-style). Do not use latest-release.

LINUX_PREBUILT_VERSION = 6.18.0-r1
LINUX_PREBUILT_SOURCE = bigfred-kernel-rpi5-v$(LINUX_PREBUILT_VERSION).tar.xz
LINUX_PREBUILT_SITE = \
	https://github.com/dcc-bigfred/hub-kernel/releases/download/v$(LINUX_PREBUILT_VERSION)
LINUX_PREBUILT_LICENSE = GPL-2.0
LINUX_PREBUILT_STRIP_COMPONENTS = 0
LINUX_PREBUILT_INSTALL_IMAGES = YES
LINUX_PREBUILT_DEPENDENCIES = host-kmod

# Tarball is Image + DTBs + overlays + lib/modules at archive root.

define LINUX_PREBUILT_INSTALL_IMAGES_CMDS
	test -f $(@D)/Image
	test -f $(@D)/bcm2712-rpi-5-b.dtb
	test -f $(@D)/bcm2712d0-rpi-5-b.dtb
	test -f $(@D)/overlays/bcm2712d0.dtbo
	$(INSTALL) -D -m 0644 $(@D)/Image $(BINARIES_DIR)/Image
	$(INSTALL) -D -m 0644 $(@D)/bcm2712-rpi-5-b.dtb \
		$(BINARIES_DIR)/bcm2712-rpi-5-b.dtb
	$(INSTALL) -D -m 0644 $(@D)/bcm2712d0-rpi-5-b.dtb \
		$(BINARIES_DIR)/bcm2712d0-rpi-5-b.dtb
	rm -rf $(BINARIES_DIR)/overlays
	mkdir -p $(BINARIES_DIR)/overlays
	cp -a $(@D)/overlays/. $(BINARIES_DIR)/overlays/
endef

define LINUX_PREBUILT_INSTALL_TARGET_CMDS
	test -f $(@D)/manifest
	test -d $(@D)/lib/modules
	krel=$$(sed -n 's/^kernelrelease=//p' $(@D)/manifest); \
	test -n "$$krel"; \
	test -d $(@D)/lib/modules/$$krel; \
	rm -rf $(TARGET_DIR)/lib/modules; \
	mkdir -p $(TARGET_DIR)/lib; \
	cp -a $(@D)/lib/modules $(TARGET_DIR)/lib/; \
	$(HOST_DIR)/sbin/depmod -a -b $(TARGET_DIR) $$krel
endef

$(eval $(generic-package))
