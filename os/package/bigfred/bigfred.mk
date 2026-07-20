################################################################################
#
# bigfred — loco-server (and dcc-bus subcommand) + bigfred-remote-icmp from
# dcc-bigfred/bigfred
#
################################################################################

BIGFRED_VERSION = $(call qstrip,$(BR2_PACKAGE_BIGFRED_VERSION))
ifeq ($(BIGFRED_VERSION),)
BIGFRED_VERSION = master
endif

BIGFRED_SITE = $(call github,dcc-bigfred,bigfred,$(BIGFRED_VERSION))
BIGFRED_LICENSE = proprietary
BIGFRED_DEPENDENCIES = host-libcap

BIGFRED_GOMOD = github.com/keskad/loco
BIGFRED_LDFLAGS = -s -w

# bigfred requires Go >= 1.25; Buildroot host-go is 1.23. Prefer docker/install-go.sh
# (/usr/local/go) or any Go 1.25+ on PATH. CGO off: modernc sqlite.
BIGFRED_GO_BIN := $(shell for g in /usr/local/go/bin/go $$(command -v go 2>/dev/null); do \
	[ -n "$$g" ] || continue; \
	[ -x "$$g" ] || g=$$(command -v "$$g" 2>/dev/null) || continue; \
	"$$g" env GOTOOLCHAIN=local GOFLAGS= go version 2>/dev/null | grep -qE 'go1\.(2[5-9]|[3-9][0-9])' && echo "$$g" && exit 0; \
	done; echo "$(HOST_DIR)/bin/go")

ifeq ($(BIGFRED_GO_BIN),$(HOST_DIR)/bin/go)
BIGFRED_GO_TOOLCHAIN = go1.25.0
else
BIGFRED_GO_TOOLCHAIN = local
endif

# golang-package appends this to download (go mod vendor) and build steps.
BIGFRED_VENDOR_GO_ENV = \
	CGO_ENABLED=0 \
	GOTOOLCHAIN=$(BIGFRED_GO_TOOLCHAIN) \
	GOFLAGS= \
	GOPROXY=https://proxy.golang.org,direct \
	GOSUMDB=sum.golang.org \
	PATH=$(patsubst %/,%,$(dir $(BIGFRED_GO_BIN))):$(PATH)

BIGFRED_BUILD_GO_ENV = \
	$(BIGFRED_VENDOR_GO_ENV) \
	GOFLAGS=-mod=vendor

BIGFRED_GO_ENV = $(BIGFRED_VENDOR_GO_ENV)

define BIGFRED_BUILD_CMDS
	cd $(@D); \
	$(BIGFRED_BUILD_GO_ENV) \
		GOOS=linux GOARCH=arm64 \
		$(BIGFRED_GO_BIN) build -v $(BIGFRED_BUILD_OPTS) \
			-o $(@D)/bin/bigfred \
			$(BIGFRED_GOMOD)/pkgs/bigfred/server
	cd $(@D); \
	$(BIGFRED_BUILD_GO_ENV) \
		GOOS=linux GOARCH=arm64 \
		$(BIGFRED_GO_BIN) build -v $(BIGFRED_BUILD_OPTS) \
			-o $(@D)/bin/bigfred-remote-icmp \
			$(BIGFRED_GOMOD)/pkgs/bigfred/remote-icmp
endef

define BIGFRED_INSTALL_TARGET_CMDS
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/opt/bigfred/bin
	$(INSTALL) -D -m 0755 $(@D)/bin/bigfred \
		$(TARGET_DIR)/opt/bigfred/bin/bigfred
	$(INSTALL) -D -m 0755 $(@D)/bin/bigfred-remote-icmp \
		$(TARGET_DIR)/opt/bigfred/bin/bigfred-remote-icmp
	$(INSTALL) -D -m 0755 $(BIGFRED_PKGDIR)/bigfred.wrapper \
		$(TARGET_DIR)/usr/bin/bigfred
	$(INSTALL) -D -m 0755 $(BIGFRED_PKGDIR)/bigfred-remote-icmp.wrapper \
		$(TARGET_DIR)/usr/bin/bigfred-remote-icmp
	# ICMP Echo probes from bigfred-remote-icmp (also covered by ping_group_range).
	$(HOST_DIR)/sbin/setcap cap_net_raw+ep $(TARGET_DIR)/opt/bigfred/bin/bigfred-remote-icmp
endef

$(eval $(golang-package))
