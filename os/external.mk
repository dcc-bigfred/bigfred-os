# SPDX-License-Identifier: MIT

# GCC 15+ defaults to -std=gnu23; several host packages in BR 2024.11 fail.
# Apply -std=gnu17 only to C autotools builds (never HOST_CFLAGS globally —
# that breaks host-gcc-initial C++11 detection).
# https://lists.buildroot.org/pipermail/buildroot/2025-May/778390.html

HOST_GCC15_C_STD = -std=gnu17
# CONF_ENV alone is not enough: autotools may not propagate CFLAGS to every sub-make.
HOST_GCC15_C_ENV = CFLAGS="$(HOST_CFLAGS) $(HOST_GCC15_C_STD)"

HOST_M4_CONF_ENV = $(HOST_GCC15_C_ENV)
HOST_M4_MAKE_ENV = $(HOST_GCC15_C_ENV)

HOST_E2FSPROGS_CONF_ENV = $(HOST_GCC15_C_ENV)
HOST_E2FSPROGS_MAKE_ENV = $(HOST_GCC15_C_ENV)

# Prebuilt binary arch mapping (Grafana, VictoriaMetrics, …)
include $(BR2_EXTERNAL_BIGFRED_HUB_PATH)/package/prebuilt-arch.mk

# BR2_EXTERNAL packages
EXTERNAL_PKG_MKS := $(sort $(wildcard $(BR2_EXTERNAL_BIGFRED_HUB_PATH)/package/*/*.mk))
include $(EXTERNAL_PKG_MKS)
