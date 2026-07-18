# SPDX-License-Identifier: MIT
# Map Buildroot target CPU to upstream prebuilt release suffixes.
# Grafana and VictoriaMetrics use different naming for 32-bit ARM.

ifeq ($(BR2_aarch64),y)
BIGFRED_PREBUILT_ARCH_GRAFANA = arm64
BIGFRED_PREBUILT_ARCH_VICTORIAMETRICS = arm64
BIGFRED_PREBUILT_ARCH_ALLOY = arm64
else ifeq ($(BR2_arm)$(BR2_armeb),y)
BIGFRED_PREBUILT_ARCH_GRAFANA = armv7
BIGFRED_PREBUILT_ARCH_VICTORIAMETRICS = arm
else
$(error BigFred prebuilt packages: unsupported target architecture $(BR2_ARCH))
endif
