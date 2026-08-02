################################################################################
#
# bigfred — loco-server + bigfred-remote-icmp from GHCR OCI bundle
# (ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64)
#
################################################################################

# Prefer Makefile override (make image BIGFRED_OCI_TAG=master), else Kconfig.
BIGFRED_VERSION = $(call qstrip,$(BIGFRED_OCI_TAG))
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = $(call qstrip,$(BR2_PACKAGE_BIGFRED_OCI_TAG))
endif
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = latest-release
endif

# Sanitize for use as a download filename (no path separators).
BIGFRED_VERSION_SAFE = $(subst /,_,$(BIGFRED_VERSION))

BIGFRED_SOURCE = bigfred-hub-linux-arm64-$(BIGFRED_VERSION_SAFE).tar
# SITE is unused for content (PRE_DOWNLOAD fills DL_DIR); keep non-empty so
# Buildroot still registers a MAIN_DOWNLOAD. After the hook creates the
# tarball, dl-wrapper finds it and skips network fetch.
BIGFRED_SITE = https://ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64
BIGFRED_LICENSE = proprietary
BIGFRED_DEPENDENCIES = host-libcap
# No bigfred.hash: floating OCI tags change under the same filename.
# dl-wrapper allows an existing DL file when no .hash is present.

# Buildroot 2025.x dropped *_DOWNLOAD_CMDS; use PRE_DOWNLOAD instead of
# wget/SITE. Floating tags are re-pulled every package download.
define BIGFRED_FETCH_OCI
	mkdir -p $(BIGFRED_DL_DIR)
	case "$(BIGFRED_VERSION)" in \
		master|latest-release|sha-*) \
			rm -f "$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)" ;; \
	esac
	$(BIGFRED_PKGDIR)/fetch-oci.sh "$(BIGFRED_VERSION)" \
		"$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)"
endef
BIGFRED_PRE_DOWNLOAD_HOOKS += BIGFRED_FETCH_OCI

define BIGFRED_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)
endef

define BIGFRED_INSTALL_TARGET_CMDS
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/opt/bigfred/bin
	$(INSTALL) -D -m 0755 $(@D)/bin/bigfred \
		$(TARGET_DIR)/opt/bigfred/bin/bigfred
	$(INSTALL) -D -m 0755 $(@D)/bin/bigfred-remote-icmp \
		$(TARGET_DIR)/opt/bigfred/bin/bigfred-remote-icmp
	$(INSTALL) -D -m 0755 $(BIGFRED_PKGDIR)/bigfred.wrapper \
		$(TARGET_DIR)/usr/bin/bigfred
	$(INSTALL) -D -m 0755 $(BIGFRED_PKGDIR)/bigfred-remote-icmp.wrapper \
		$(TARGET_DIR)/usr/bin/bigfred-remote-icmp
	# ICMP Echo probes from bigfred-remote-icmp (also covered by ping_group_range).
	$(HOST_DIR)/sbin/setcap cap_net_raw+ep $(TARGET_DIR)/opt/bigfred/bin/bigfred-remote-icmp
endef

$(eval $(generic-package))
