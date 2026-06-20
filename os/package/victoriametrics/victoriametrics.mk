# SPDX-License-Identifier: MIT
# VictoriaMetrics single-node (prebuilt binary from GitHub releases; arch from prebuilt-arch.mk).

VICTORIAMETRICS_VERSION = 1.130.0
VICTORIAMETRICS_SOURCE = victoria-metrics-linux-$(BIGFRED_PREBUILT_ARCH_VICTORIAMETRICS)-v$(VICTORIAMETRICS_VERSION).tar.gz
VICTORIAMETRICS_SITE = \
	https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/v$(VICTORIAMETRICS_VERSION)
VICTORIAMETRICS_LICENSE = Apache-2.0

define VICTORIAMETRICS_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/victoria-metrics-prod \
		$(TARGET_DIR)/usr/bin/victoria-metrics
endef

$(eval $(generic-package))
