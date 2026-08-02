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
BIGFRED_SITE =
BIGFRED_LICENSE = proprietary
BIGFRED_DEPENDENCIES = host-libcap

# Floating tags change under the same name — always re-pull.
define BIGFRED_DOWNLOAD_CMDS
	case "$(BIGFRED_VERSION)" in \
		master|latest-release|sha-*) \
			rm -f "$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)" ;; \
	esac
	$(BIGFRED_PKGDIR)/fetch-oci.sh "$(BIGFRED_VERSION)" \
		"$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)"
endef

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
