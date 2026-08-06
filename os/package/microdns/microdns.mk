################################################################################
#
# microdns — mDNS/DNS-SD advertiser from GHCR OCI bundle
# (ghcr.io/dcc-bigfred/microdns-linux-arm64)
#
################################################################################

# Prefer Makefile override (make image MICRODNS_OCI_TAG=main), else Kconfig.
MICRODNS_VERSION = $(call qstrip,$(MICRODNS_OCI_TAG))
ifeq ($(MICRODNS_VERSION),)
MICRODNS_VERSION = $(call qstrip,$(BR2_PACKAGE_MICRODNS_OCI_TAG))
endif
ifeq ($(MICRODNS_VERSION),)
MICRODNS_VERSION = main
endif

# Sanitize for use as a download filename (no path separators).
MICRODNS_VERSION_SAFE = $(subst /,_,$(MICRODNS_VERSION))

MICRODNS_SOURCE = microdns-linux-arm64-$(MICRODNS_VERSION_SAFE).tar
# SITE is unused for content (PRE_DOWNLOAD fills DL_DIR); keep non-empty so
# Buildroot still registers a MAIN_DOWNLOAD.
MICRODNS_SITE = https://ghcr.io/dcc-bigfred/microdns-linux-arm64
MICRODNS_LICENSE = MIT
# No microdns.hash: floating OCI tags change under the same filename.

# Always re-fetch the OCI artifact: even pinned tags can be re-pushed, and
# floating tags (main/latest-release) change under the same filename.
define MICRODNS_FETCH_OCI
	mkdir -p $(MICRODNS_DL_DIR)
	rm -f "$(MICRODNS_DL_DIR)/$(MICRODNS_SOURCE)"
	$(MICRODNS_PKGDIR)/fetch-oci.sh "$(MICRODNS_VERSION)" \
		"$(MICRODNS_DL_DIR)/$(MICRODNS_SOURCE)"
endef
MICRODNS_PRE_DOWNLOAD_HOOKS += MICRODNS_FETCH_OCI

define MICRODNS_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICRODNS_DL_DIR)/$(MICRODNS_SOURCE)
endef

define MICRODNS_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/microdns \
		$(TARGET_DIR)/usr/sbin/microdns
endef

$(eval $(generic-package))

# Force the download step to re-run on every `make image`: without this,
# .stamp_downloaded stays present after the first build and Buildroot skips
# PRE_DOWNLOAD_HOOKS, so a re-pushed OCI tag would never be re-fetched.
.PHONY: MICRODNS_FORCE_REDOWNLOAD
$(MICRODNS_DIR)/.stamp_downloaded: MICRODNS_FORCE_REDOWNLOAD
