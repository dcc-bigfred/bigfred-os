################################################################################
#
# wireless-programmer — device discovery + programming daemon from GitHub
# (Actions tip / Releases)
#
################################################################################

WIRELESS_PROGRAMMER_VERSION = $(call qstrip,$(WIRELESS_PROGRAMMER_REF))
ifeq ($(WIRELESS_PROGRAMMER_VERSION),)
WIRELESS_PROGRAMMER_VERSION = $(call qstrip,$(BR2_PACKAGE_WIRELESS_PROGRAMMER_REF))
endif
ifeq ($(WIRELESS_PROGRAMMER_VERSION),)
WIRELESS_PROGRAMMER_VERSION = main
endif

WIRELESS_PROGRAMMER_VERSION_SAFE = $(subst /,_,$(WIRELESS_PROGRAMMER_VERSION))

WIRELESS_PROGRAMMER_SOURCE = wireless-programmer-linux-arm64-$(WIRELESS_PROGRAMMER_VERSION_SAFE).tar
WIRELESS_PROGRAMMER_SITE = https://github.com/dcc-bigfred/wireless-programmer
WIRELESS_PROGRAMMER_LICENSE = MIT

define WIRELESS_PROGRAMMER_FETCH_GITHUB
	mkdir -p $(WIRELESS_PROGRAMMER_DL_DIR)
	rm -f "$(WIRELESS_PROGRAMMER_DL_DIR)/$(WIRELESS_PROGRAMMER_SOURCE)"
	$(WIRELESS_PROGRAMMER_PKGDIR)/fetch.sh "$(WIRELESS_PROGRAMMER_VERSION)" \
		"$(WIRELESS_PROGRAMMER_DL_DIR)/$(WIRELESS_PROGRAMMER_SOURCE)"
endef
WIRELESS_PROGRAMMER_PRE_DOWNLOAD_HOOKS += WIRELESS_PROGRAMMER_FETCH_GITHUB

define WIRELESS_PROGRAMMER_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(WIRELESS_PROGRAMMER_DL_DIR)/$(WIRELESS_PROGRAMMER_SOURCE)
endef

define WIRELESS_PROGRAMMER_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/wireless-programmer-linux-arm64 \
		$(TARGET_DIR)/usr/sbin/wireless-programmer
endef

$(eval $(generic-package))

# Force the download step to re-run on every `make image`: without this,
# .stamp_downloaded stays present after the first build and Buildroot skips
# PRE_DOWNLOAD_HOOKS, so a re-pushed tip would never be re-fetched.
.PHONY: WIRELESS_PROGRAMMER_FORCE_REDOWNLOAD
$(WIRELESS_PROGRAMMER_DIR)/.stamp_downloaded: WIRELESS_PROGRAMMER_FORCE_REDOWNLOAD
