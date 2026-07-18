# SPDX-License-Identifier: MIT
# Grafana Alloy (prebuilt zip from GitHub releases; arch from prebuilt-arch.mk).

ALLOY_VERSION = 1.17.1
ALLOY_SOURCE = alloy-linux-$(BIGFRED_PREBUILT_ARCH_ALLOY).zip
ALLOY_SITE = https://github.com/grafana/alloy/releases/download/v$(ALLOY_VERSION)
ALLOY_LICENSE = Apache-2.0

define ALLOY_EXTRACT_CMDS
	$(UNZIP) -d $(@D) $(ALLOY_DL_DIR)/$(ALLOY_SOURCE)
endef

define ALLOY_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/alloy-linux-$(BIGFRED_PREBUILT_ARCH_ALLOY) \
		$(TARGET_DIR)/usr/bin/alloy
endef

$(eval $(generic-package))
