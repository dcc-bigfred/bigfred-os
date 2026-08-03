################################################################################
#
# microinit — PID 1 / service supervisor from GHCR OCI bundle
# (ghcr.io/dcc-bigfred/microinit-linux-arm64)
#
################################################################################

# Prefer Makefile override (make image MICROINIT_OCI_TAG=main), else Kconfig.
MICROINIT_VERSION = $(call qstrip,$(MICROINIT_OCI_TAG))
ifeq ($(MICROINIT_VERSION),)
MICROINIT_VERSION = $(call qstrip,$(BR2_PACKAGE_MICROINIT_OCI_TAG))
endif
ifeq ($(MICROINIT_VERSION),)
MICROINIT_VERSION = main
endif

# Sanitize for use as a download filename (no path separators).
MICROINIT_VERSION_SAFE = $(subst /,_,$(MICROINIT_VERSION))

MICROINIT_SOURCE = microinit-linux-arm64-$(MICROINIT_VERSION_SAFE).tar
# SITE is unused for content (PRE_DOWNLOAD fills DL_DIR); keep non-empty so
# Buildroot still registers a MAIN_DOWNLOAD.
MICROINIT_SITE = https://ghcr.io/dcc-bigfred/microinit-linux-arm64
MICROINIT_LICENSE = proprietary
# No microinit.hash: floating OCI tags change under the same filename.

# Always re-fetch the OCI artifact: even pinned tags can be re-pushed, and
# floating tags (main/latest-release) change under the same filename. There
# is no microinit.hash, so Buildroot cannot detect content drift on its own.
define MICROINIT_FETCH_OCI
	mkdir -p $(MICROINIT_DL_DIR)
	rm -f "$(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)"
	$(MICROINIT_PKGDIR)/fetch-oci.sh "$(MICROINIT_VERSION)" \
		"$(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)"
endef
MICROINIT_PRE_DOWNLOAD_HOOKS += MICROINIT_FETCH_OCI

define MICROINIT_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)
endef

define MICROINIT_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/microinit \
		$(TARGET_DIR)/usr/sbin/microinit
	# PID 1: kernel default /sbin/init
	$(INSTALL) -D -m 0755 $(@D)/bin/microinit \
		$(TARGET_DIR)/sbin/init
endef

$(eval $(generic-package))

# Force the download step to re-run on every `make image`: without this,
# .stamp_downloaded stays present after the first build and Buildroot skips
# PRE_DOWNLOAD_HOOKS, so a re-pushed OCI tag would never be re-fetched.
.PHONY: MICROINIT_FORCE_REDOWNLOAD
$(MICROINIT_DIR)/.stamp_downloaded: MICROINIT_FORCE_REDOWNLOAD
