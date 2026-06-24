# BigFred hub OS — Buildroot wrapper
# Requires: host packages per https://buildroot.org/downloads/manual/manual.html#requirement

BUILDROOT_VERSION ?= 2025.02
BUILDROOT_URL       = https://github.com/buildroot/buildroot/archive/$(BUILDROOT_VERSION).tar.gz

CURDIR            := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
REPO_ROOT         := $(abspath $(CURDIR)/..)
BR2_EXTERNAL      := $(CURDIR)
BUILDROOT_DIR     ?= $(CURDIR)/buildroot
OUTPUT_DIR        ?= $(CURDIR)/output
DEFCONFIG         ?= bigfred_hub_rpi5_defconfig
APPS_BIN_DIR      := $(REPO_ROOT)/apps/.bin

BUILDROOT_TAR     = $(CURDIR)/buildroot-$(BUILDROOT_VERSION).tar.gz
BR2_MAKE          = $(MAKE) -C $(BUILDROOT_DIR) O=$(OUTPUT_DIR) BR2_EXTERNAL=$(BR2_EXTERNAL)

.PHONY: all help buildroot defconfig menuconfig image legal-info clean distclean flash apps-build apps-clean

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
	@echo "  make apps-build  — cross-compile Go apps to apps/.bin/"
	@echo "  make image       — apps-build + kernel, rootfs, hub-nvme.img"
	@echo "  make clean       — clean build output (keeps download cache)"
	@echo "  make distclean   — remove output/ and buildroot/"
	@echo ""
	@echo "Artifact: $(OUTPUT_DIR)/images/hub-nvme.img"
	@echo "BigFred binaries are NOT built here — install separately."

$(BUILDROOT_DIR)/Makefile: $(BUILDROOT_TAR)
	rm -rf $(BUILDROOT_DIR)
	mkdir -p $(BUILDROOT_DIR)
	tar -xf $(BUILDROOT_TAR) -C $(BUILDROOT_DIR) --strip-components=1
	touch $@

$(BUILDROOT_TAR):
	curl -fL "$(BUILDROOT_URL)" -o "$(BUILDROOT_TAR)"

buildroot: $(BUILDROOT_DIR)/Makefile

defconfig: buildroot
	$(BR2_MAKE) $(DEFCONFIG)

menuconfig: buildroot
	$(BR2_MAKE) menuconfig

image: defconfig
	$(BR2_MAKE) linux-pam
	$(MAKE) -C $(REPO_ROOT)/apps build
	$(BR2_MAKE)
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
