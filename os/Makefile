# BigFred hub OS — Buildroot wrapper
# Requires: host packages per https://buildroot.org/downloads/manual/manual.html#requirement
# Binary fetch uses: go run github.com/dcc-bigfred/common/cmd/fetch@latest

BUILDROOT_VERSION ?= 2025.02
BUILDROOT_URL       = https://github.com/buildroot/buildroot/archive/$(BUILDROOT_VERSION).tar.gz

CURDIR            := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
REPO_ROOT         := $(abspath $(CURDIR)/..)
BR2_EXTERNAL      := $(CURDIR)
BUILDROOT_DIR     ?= $(CURDIR)/buildroot
OUTPUT_DIR        ?= $(CURDIR)/output
DEFCONFIG         ?= bigfred_hub_rpi5_defconfig
APPS_BIN_DIR      := $(REPO_ROOT)/apps/.bin

# Defaults from repo-root VERSIONS; CLI/env still override.
include $(REPO_ROOT)/VERSIONS

BUILDROOT_TAR     = $(CURDIR)/buildroot-$(BUILDROOT_VERSION).tar.gz
BR2_MAKE          = $(MAKE) -C $(BUILDROOT_DIR) O=$(OUTPUT_DIR) BR2_EXTERNAL=$(BR2_EXTERNAL)

.PHONY: all help buildroot defconfig menuconfig image legal-info clean distclean flash \
	apps-build apps-clean

all: image

apps-build:
	$(MAKE) -C $(REPO_ROOT)/apps build

apps-clean:
	$(MAKE) -C $(REPO_ROOT)/apps clean

help:
	@echo "BigFred hub OS (Raspberry Pi 5)"
	@echo ""
	@echo "  make buildroot   — download and extract Buildroot $(BUILDROOT_VERSION)"
	@echo "  make defconfig   — load $(DEFCONFIG)"
	@echo "  make menuconfig  — customize Buildroot"
	@echo "  make apps-build  — build every apps/*/Makefile → apps/.bin/"
	@echo "  make image       — apps-build + kernel, rootfs, hub-nvme.img"
	@echo "  make clean       — clean build output (keeps download cache)"
	@echo "  make distclean   — remove output/ and buildroot/"
	@echo ""
	@echo "Refs from $(REPO_ROOT)/VERSIONS (override: make image BIGFRED_REF=…)"
	@echo "  BigFred:    BIGFRED_REF=$(BIGFRED_REF) (master|latest-release|v*|sha-*)"
	@echo "  microinit:  MICROINIT_REF=$(MICROINIT_REF) (main|sha-*|v*|latest-release)"
	@echo "  micronet:   MICRONET_REF=$(MICRONET_REF) (main|sha-*|v*|latest-release)"
	@echo "  microdns:   MICRODNS_REF=$(MICRODNS_REF) (main|sha-*|v*|latest-release)"
	@echo "Tip refs need GITHUB_TOKEN / GH_TOKEN / BIGFRED_NATIVE_TOKEN"
	@echo "Binary fetch: go run github.com/dcc-bigfred/common/cmd/fetch@latest"

$(BUILDROOT_DIR)/Makefile: $(BUILDROOT_TAR)
	rm -rf $(BUILDROOT_DIR)
	mkdir -p $(BUILDROOT_DIR)
	tar -xf $(BUILDROOT_TAR) -C $(BUILDROOT_DIR) --strip-components=1
	touch $@

$(BUILDROOT_TAR):
	curl -fL "$(BUILDROOT_URL)" -o "$(BUILDROOT_TAR)"

buildroot: $(BUILDROOT_DIR)/Makefile

defconfig: buildroot $(CURDIR)/configs/$(DEFCONFIG)
	$(BR2_MAKE) $(DEFCONFIG)

menuconfig: buildroot
	$(BR2_MAKE) menuconfig

image: defconfig
	@command -v go >/dev/null 2>&1 || { echo "error: go required for package/*/fetch.sh"; exit 1; }
	$(BR2_MAKE) linux-pam
	# BIGFRED_PERMISSIONS uses |xattr; host-makedevs must be built with
	# -DEXTENDED_ATTRIBUTES. Rebuild when the binary still lacks that support
	# (stale from before BR2_ROOTFS_DEVICE_TABLE_SUPPORTS_EXTENDED_ATTRIBUTES=y).
	@if grep -q '^BR2_ROOTFS_DEVICE_TABLE_SUPPORTS_EXTENDED_ATTRIBUTES=y' \
		"$(OUTPUT_DIR)/.config" 2>/dev/null; then \
		if [ ! -x "$(OUTPUT_DIR)/host/bin/makedevs" ] || \
		   strings "$(OUTPUT_DIR)/host/bin/makedevs" 2>/dev/null | \
			grep -q 'Did you forget to enable BR2_ROOTFS_DEVICE_TABLE'; then \
			echo "Rebuilding host-makedevs with extended attribute support…"; \
			$(BR2_MAKE) host-makedevs-rebuild; \
		fi; \
	fi
	$(MAKE) -C $(REPO_ROOT)/apps build
	$(BR2_MAKE) BIGFRED_REF=$(BIGFRED_REF) MICROINIT_REF=$(MICROINIT_REF) MICRONET_REF=$(MICRONET_REF) MICRODNS_REF=$(MICRODNS_REF)
	@echo ""
	@echo "Image ready: $(OUTPUT_DIR)/images/hub-nvme.img"
	@ls -lh "$(OUTPUT_DIR)/images/hub-nvme.img" 2>/dev/null || true

legal-info: buildroot
	$(BR2_MAKE) legal-info

clean: buildroot apps-clean
	$(BR2_MAKE) clean

distclean:
	rm -rf "$(OUTPUT_DIR)" "$(BUILDROOT_DIR)" "$(BUILDROOT_TAR)"

flash:
	@echo "Run on host with NVMe attached, e.g.:"
	@echo "  sudo $(CURDIR)/scripts/flash-nvme.sh /dev/nvme0n1 $(OUTPUT_DIR)/images/hub-nvme.img"
