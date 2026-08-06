################################################################################
#
# micronet — configure-ethernet + configure-dhcp from GHCR OCI bundle
# (ghcr.io/dcc-bigfred/micronet-linux-arm64)
#
################################################################################

MICRONET_VERSION = $(call qstrip,$(MICRONET_OCI_TAG))
ifeq ($(MICRONET_VERSION),)
MICRONET_VERSION = $(call qstrip,$(BR2_PACKAGE_MICRONET_OCI_TAG))
endif
ifeq ($(MICRONET_VERSION),)
MICRONET_VERSION = main
endif

MICRONET_VERSION_SAFE = $(subst /,_,$(MICRONET_VERSION))

MICRONET_SOURCE = micronet-linux-arm64-$(MICRONET_VERSION_SAFE).tar
MICRONET_SITE = https://ghcr.io/dcc-bigfred/micronet-linux-arm64
MICRONET_LICENSE = MIT

define MICRONET_FETCH_OCI
	mkdir -p $(MICRONET_DL_DIR)
	rm -f "$(MICRONET_DL_DIR)/$(MICRONET_SOURCE)"
	$(MICRONET_PKGDIR)/fetch-oci.sh "$(MICRONET_VERSION)" \
		"$(MICRONET_DL_DIR)/$(MICRONET_SOURCE)"
endef
MICRONET_PRE_DOWNLOAD_HOOKS += MICRONET_FETCH_OCI

define MICRONET_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICRONET_DL_DIR)/$(MICRONET_SOURCE)
endef

define MICRONET_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/configure-ethernet \
		$(TARGET_DIR)/usr/sbin/configure-ethernet
	$(INSTALL) -D -m 0755 $(@D)/bin/configure-dhcp \
		$(TARGET_DIR)/usr/sbin/configure-dhcp
endef

$(eval $(generic-package))

.PHONY: MICRONET_FORCE_REDOWNLOAD
$(MICRONET_DIR)/.stamp_downloaded: MICRONET_FORCE_REDOWNLOAD
