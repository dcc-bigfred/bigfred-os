################################################################################
#
# micronet — network daemon from GitHub (Actions tip / Releases)
#
################################################################################

MICRONET_VERSION = $(call qstrip,$(MICRONET_REF))
ifeq ($(MICRONET_VERSION),)
MICRONET_VERSION = $(call qstrip,$(BR2_PACKAGE_MICRONET_REF))
endif
ifeq ($(MICRONET_VERSION),)
MICRONET_VERSION = main
endif

MICRONET_VERSION_SAFE = $(subst /,_,$(MICRONET_VERSION))

MICRONET_SOURCE = micronet-linux-arm64-$(MICRONET_VERSION_SAFE).tar
MICRONET_SITE = https://github.com/dcc-bigfred/micronet
MICRONET_LICENSE = MIT
MICRONET_DEPENDENCIES = dnsmasq dhcp iproute2

define MICRONET_FETCH_GITHUB
	mkdir -p $(MICRONET_DL_DIR)
	rm -f "$(MICRONET_DL_DIR)/$(MICRONET_SOURCE)"
	$(MICRONET_PKGDIR)/fetch.sh "$(MICRONET_VERSION)" \
		"$(MICRONET_DL_DIR)/$(MICRONET_SOURCE)"
endef
MICRONET_PRE_DOWNLOAD_HOOKS += MICRONET_FETCH_GITHUB

define MICRONET_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICRONET_DL_DIR)/$(MICRONET_SOURCE)
endef

define MICRONET_INSTALL_TARGET_CMDS
	test -x $(@D)/bin/micronet
	$(INSTALL) -D -m 0755 $(@D)/bin/micronet \
		$(TARGET_DIR)/usr/sbin/micronet
	ln -sf micronet $(TARGET_DIR)/usr/sbin/configure-ethernet
	ln -sf micronet $(TARGET_DIR)/usr/sbin/configure-dhcp
endef

$(eval $(generic-package))

.PHONY: MICRONET_FORCE_REDOWNLOAD
$(MICRONET_DIR)/.stamp_downloaded: MICRONET_FORCE_REDOWNLOAD
