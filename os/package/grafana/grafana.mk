# SPDX-License-Identifier: MIT
# Grafana OSS (prebuilt tarball from dl.grafana.com; arch from prebuilt-arch.mk).
#
# Install the unified static `grafana` binary (`grafana server` / `grafana cli`).
# Older images shipped grafana-server + grafana-cli (~2 MiB wrappers); the hub
# init script expects /usr/bin/grafana.

GRAFANA_VERSION = 11.6.1
GRAFANA_SOURCE = grafana-$(GRAFANA_VERSION).linux-$(BIGFRED_PREBUILT_ARCH_GRAFANA).tar.gz
GRAFANA_SITE = https://dl.grafana.com/oss/release
GRAFANA_LICENSE = AGPL-3.0

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
endef

$(eval $(generic-package))
