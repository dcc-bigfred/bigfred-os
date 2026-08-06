################################################################################
#
# microinit — PID 1 / service supervisor from GitHub
# (Actions tip / Releases; not the distroless container image)
#
################################################################################

MICROINIT_VERSION = $(call qstrip,$(MICROINIT_OCI_TAG))
ifeq ($(MICROINIT_VERSION),)
MICROINIT_VERSION = $(call qstrip,$(BR2_PACKAGE_MICROINIT_OCI_TAG))
endif
ifeq ($(MICROINIT_VERSION),)
MICROINIT_VERSION = main
endif

MICROINIT_VERSION_SAFE = $(subst /,_,$(MICROINIT_VERSION))

MICROINIT_SOURCE = microinit-linux-arm64-$(MICROINIT_VERSION_SAFE).tar
MICROINIT_SITE = https://github.com/dcc-bigfred/microinit
MICROINIT_LICENSE = proprietary

define MICROINIT_FETCH_GITHUB
	mkdir -p $(MICROINIT_DL_DIR)
	rm -f "$(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)"
	$(MICROINIT_PKGDIR)/fetch.sh "$(MICROINIT_VERSION)" \
		"$(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)"
endef
MICROINIT_PRE_DOWNLOAD_HOOKS += MICROINIT_FETCH_GITHUB

define MICROINIT_EXTRACT_CMDS
	$(TAR) -C $(@D) -xf $(MICROINIT_DL_DIR)/$(MICROINIT_SOURCE)
endef

define MICROINIT_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/microinit \
		$(TARGET_DIR)/usr/sbin/microinit
	$(INSTALL) -D -m 0755 $(@D)/bin/microinit \
		$(TARGET_DIR)/sbin/init
	if [ -f $(@D)/bin/shutdown ]; then \
		$(INSTALL) -D -m 0755 $(@D)/bin/shutdown \
			$(TARGET_DIR)/usr/sbin/shutdown; \
		mkdir -p $(TARGET_DIR)/sbin; \
		ln -sfn ../usr/sbin/shutdown $(TARGET_DIR)/sbin/shutdown; \
	fi
endef

$(eval $(generic-package))

.PHONY: MICROINIT_FORCE_REDOWNLOAD
$(MICROINIT_DIR)/.stamp_downloaded: MICROINIT_FORCE_REDOWNLOAD
