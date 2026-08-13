################################################################################
#
# microwaf — host-side WAF from GitHub (Actions tip / Releases)
#
################################################################################

MICROWAF_VERSION = $(call qstrip,$(MICROWAF_REF))
ifeq ($(MICROWAF_VERSION),)
MICROWAF_VERSION = $(call qstrip,$(BR2_PACKAGE_MICROWAF_REF))
endif
ifeq ($(MICROWAF_VERSION),)
MICROWAF_VERSION = main
endif

MICROWAF_VERSION_SAFE = $(subst /,_,$(MICROWAF_VERSION))

MICROWAF_SOURCE = microwaf-linux-arm64-$(MICROWAF_VERSION_SAFE).tar
MICROWAF_SITE = https://github.com/dcc-bigfred/microwaf
MICROWAF_LICENSE = MIT

define MICROWAF_FETCH_GITHUB
	mkdir -p $(MICROWAF_DL_DIR)
	rm -f "$(MICROWAF_DL_DIR)/$(MICROWAF_SOURCE)"
	$(MICROWAF_PKGDIR)/fetch.sh "$(MICROWAF_VERSION)" \
		"$(MICROWAF_DL_DIR)/$(MICROWAF_SOURCE)"
endef
MICROWAF_PRE_DOWNLOAD_HOOKS += MICROWAF_FETCH_GITHUB

define MICROWAF_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICROWAF_DL_DIR)/$(MICROWAF_SOURCE)
endef

define MICROWAF_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/microwaf-linux-arm64 \
		$(TARGET_DIR)/usr/sbin/microwaf
endef

$(eval $(generic-package))

# Force the download step to re-run on every `make image`: without this,
# .stamp_downloaded stays present after the first build and Buildroot skips
# PRE_DOWNLOAD_HOOKS, so a re-pushed tip would never be re-fetched.
.PHONY: MICROWAF_FORCE_REDOWNLOAD
$(MICROWAF_DIR)/.stamp_downloaded: MICROWAF_FORCE_REDOWNLOAD
