# SPDX-License-Identifier: MIT
# Grafana OSS (prebuilt tarball from dl.grafana.com; arch from prebuilt-arch.mk).

GRAFANA_VERSION = 11.6.1
GRAFANA_SOURCE = grafana-$(GRAFANA_VERSION).linux-$(BIGFRED_PREBUILT_ARCH_GRAFANA).tar.gz
GRAFANA_SITE = https://dl.grafana.com/oss/release
GRAFANA_LICENSE = AGPL-3.0

define GRAFANA_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/grafana-server \
		$(TARGET_DIR)/usr/bin/grafana-server
	$(INSTALL) -D -m 0755 $(@D)/bin/grafana-cli \
		$(TARGET_DIR)/usr/bin/grafana-cli
	mkdir -p $(TARGET_DIR)/usr/share/grafana
	cp -a $(@D)/conf $(@D)/public $(@D)/tools \
		$(TARGET_DIR)/usr/share/grafana/
endef

$(eval $(generic-package))
