################################################################################
#
# bigfred-wizard — party kiosk from GitHub (Actions tip / Releases)
#
################################################################################

BIGFRED_WIZARD_VERSION = $(call qstrip,$(BIGFRED_WIZARD_REF))
ifeq ($(BIGFRED_WIZARD_VERSION),)
BIGFRED_WIZARD_VERSION = $(call qstrip,$(BR2_PACKAGE_BIGFRED_WIZARD_REF))
endif
ifeq ($(BIGFRED_WIZARD_VERSION),)
BIGFRED_WIZARD_VERSION = main
endif

BIGFRED_WIZARD_VERSION_SAFE = $(subst /,_,$(BIGFRED_WIZARD_VERSION))

BIGFRED_WIZARD_SOURCE = bigfred-wizard-linux-arm64-$(BIGFRED_WIZARD_VERSION_SAFE).tar
BIGFRED_WIZARD_SITE = https://github.com/dcc-bigfred/bigfred-wizard
BIGFRED_WIZARD_LICENSE = MIT

define BIGFRED_WIZARD_FETCH_GITHUB
	mkdir -p $(BIGFRED_WIZARD_DL_DIR)
	rm -f "$(BIGFRED_WIZARD_DL_DIR)/$(BIGFRED_WIZARD_SOURCE)"
	$(BIGFRED_WIZARD_PKGDIR)/fetch.sh "$(BIGFRED_WIZARD_VERSION)" \
		"$(BIGFRED_WIZARD_DL_DIR)/$(BIGFRED_WIZARD_SOURCE)"
endef
BIGFRED_WIZARD_PRE_DOWNLOAD_HOOKS += BIGFRED_WIZARD_FETCH_GITHUB

define BIGFRED_WIZARD_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(BIGFRED_WIZARD_DL_DIR)/$(BIGFRED_WIZARD_SOURCE)
endef

define BIGFRED_WIZARD_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bigfred-wizard-linux-arm64 \
		$(TARGET_DIR)/usr/sbin/bigfred-wizard
endef

$(eval $(generic-package))

.PHONY: BIGFRED_WIZARD_FORCE_REDOWNLOAD
$(BIGFRED_WIZARD_DIR)/.stamp_downloaded: BIGFRED_WIZARD_FORCE_REDOWNLOAD
