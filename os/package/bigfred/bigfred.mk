################################################################################
#
# bigfred — loco-server (and dcc-bus subcommand) from dcc-bigfred/bigfred
#
################################################################################

BIGFRED_VERSION = $(call qstrip,$(BR2_PACKAGE_BIGFRED_VERSION))
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = master
endif

BIGFRED_SITE = $(call github,dcc-bigfred,bigfred,$(BIGFRED_VERSION))
BIGFRED_LICENSE = proprietary

BIGFRED_GOMOD = github.com/keskad/loco
BIGFRED_BUILD_TARGETS = pkgs/bigfred/server
BIGFRED_BIN_NAME = bigfred
BIGFRED_LDFLAGS = -s -w

# bigfred requires Go >= 1.25; Buildroot's host-go may be older — let the
# toolchain auto-download. CGO off: pure-Go sqlite (modernc.org/sqlite).
BIGFRED_GO_ENV = \
	CGO_ENABLED=0 \
	GOTOOLCHAIN=go1.25.0 \
	GOPROXY=https://proxy.golang.org,direct

define BIGFRED_INSTALL_TARGET_CMDS
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/opt/bigfred/bin
	$(INSTALL) -D -m 0755 $(@D)/bin/bigfred \
		$(TARGET_DIR)/opt/bigfred/bin/bigfred
	$(INSTALL) -D -m 0755 $(BIGFRED_PKGDIR)/bigfred.wrapper \
		$(TARGET_DIR)/usr/bin/bigfred
endef

$(eval $(golang-package))
