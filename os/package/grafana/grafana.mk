# SPDX-License-Identifier: MIT
# Grafana OSS (prebuilt tarball from dl.grafana.com; arch from prebuilt-arch.mk).
#
# Install the unified static `grafana` binary (`grafana server` / `grafana cli`).
# Older images shipped grafana-server + grafana-cli (~2 MiB wrappers); the hub
# init script expects /usr/bin/grafana.
#
# Dashboard JSON is fetched at install time from GitHub (BIGFRED_REF,
# MICROINIT_REF) via fetch-dashboards.sh; OSS tarball stays hash-pinned.

GRAFANA_VERSION = 11.6.1
GRAFANA_SOURCE = grafana-$(GRAFANA_VERSION).linux-$(BIGFRED_PREBUILT_ARCH_GRAFANA).tar.gz
GRAFANA_SITE = https://dl.grafana.com/oss/release
GRAFANA_LICENSE = AGPL-3.0

GRAFANA_BIGFRED_REF = $(call qstrip,$(BIGFRED_REF))
ifeq ($(GRAFANA_BIGFRED_REF),)
GRAFANA_BIGFRED_REF = $(call qstrip,$(BR2_PACKAGE_BIGFRED_REF))
endif
ifeq ($(GRAFANA_BIGFRED_REF),)
GRAFANA_BIGFRED_REF = master
endif

GRAFANA_MICROINIT_REF = $(call qstrip,$(MICROINIT_REF))
ifeq ($(GRAFANA_MICROINIT_REF),)
GRAFANA_MICROINIT_REF = $(call qstrip,$(BR2_PACKAGE_MICROINIT_REF))
endif
ifeq ($(GRAFANA_MICROINIT_REF),)
GRAFANA_MICROINIT_REF = main
endif

define GRAFANA_INSTALL_TARGET_CMDS
	# Drop legacy helpers if a previous package install left them behind.
	rm -f $(TARGET_DIR)/usr/bin/grafana-server \
		$(TARGET_DIR)/usr/bin/grafana-cli
	test -f $(@D)/bin/grafana
	$(INSTALL) -D -m 0755 $(@D)/bin/grafana \
		$(TARGET_DIR)/usr/bin/grafana
	# Fail the build if we somehow installed a non-static / missing binary.
	test -x $(TARGET_DIR)/usr/bin/grafana
	mkdir -p $(TARGET_DIR)/usr/share/grafana
	cp -a $(@D)/conf $(@D)/public $(@D)/tools \
		$(TARGET_DIR)/usr/share/grafana/
	$(GRAFANA_PKGDIR)/fetch-dashboards.sh \
		"$(GRAFANA_BIGFRED_REF)" "$(GRAFANA_MICROINIT_REF)" "$(@D)/dashboards"
	# Replace, don't merge: a removed upstream dashboard must leave the image.
	rm -rf $(TARGET_DIR)/usr/share/grafana/dashboards
	mkdir -p $(TARGET_DIR)/usr/share/grafana/dashboards
	cp -a $(@D)/dashboards/bigfred $(@D)/dashboards/microinit \
		$(TARGET_DIR)/usr/share/grafana/dashboards/
	chmod -R a+rX $(TARGET_DIR)/usr/share/grafana/dashboards
endef

$(eval $(generic-package))

# Re-run install on every `make image` so tip-ref dashboard JSON is refreshed
# without re-downloading the OSS tarball (grafana.hash stays valid).
.PHONY: GRAFANA_FORCE_REINSTALL
$(GRAFANA_TARGET_INSTALL_TARGET): GRAFANA_FORCE_REINSTALL
