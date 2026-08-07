################################################################################
#
# bigfred — loco-server + bigfred-remote-icmp from GitHub
# (Actions tip / Releases)
#
################################################################################

BIGFRED_VERSION = $(call qstrip,$(BIGFRED_REF))
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = $(call qstrip,$(BR2_PACKAGE_BIGFRED_REF))
endif
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = latest-release
endif

BIGFRED_VERSION_SAFE = $(subst /,_,$(BIGFRED_VERSION))

BIGFRED_SOURCE = bigfred-hub-linux-arm64-$(BIGFRED_VERSION_SAFE).tar
BIGFRED_SITE = https://github.com/dcc-bigfred/bigfred
BIGFRED_LICENSE = proprietary
BIGFRED_DEPENDENCIES = host-libcap

define BIGFRED_FETCH_GITHUB
	mkdir -p $(BIGFRED_DL_DIR)
	case "$(BIGFRED_VERSION)" in \
		master|main|latest-release|sha-*) \
			rm -f "$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)" ;; \
	esac
	$(BIGFRED_PKGDIR)/fetch.sh "$(BIGFRED_VERSION)" \
		"$(BIGFRED_DL_DIR)/$(BIGFRED_SOURCE)"
endef
BIGFRED_PRE_DOWNLOAD_HOOKS += BIGFRED_FETCH_GITHUB

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
endef

define BIGFRED_PERMISSIONS
	/opt/bigfred/bin/bigfred-remote-icmp f 755 0 0 - - - - -
	|xattr cap_net_raw+ep
endef

$(eval $(generic-package))
