################################################################################
#
# supervisord — ochinchina/supervisord + Python-compatible supervisorctl shim
#
# Installed as /usr/bin/supervisord and /usr/bin/supervisorctl so loco-server
# and bigfred-os-ui stay unaware of the ochinchina implementation.
#
################################################################################

SUPERVISORD_VERSION = $(call qstrip,$(BR2_PACKAGE_SUPERVISORD_VERSION))
ifeq ($(SUPERVISORD_VERSION),)
SUPERVISORD_VERSION = 0.7.3
endif

SUPERVISORD_SITE = $(call github,ochinchina,supervisord,v$(SUPERVISORD_VERSION))
SUPERVISORD_LICENSE = Apache-2.0
SUPERVISORD_LICENSE_FILES = LICENSE

# Prefer Go from docker/install-go.sh (same pattern as package/bigfred).
SUPERVISORD_GO_BIN := $(shell for g in /usr/local/go/bin/go $$(command -v go 2>/dev/null); do \
	[ -n "$$g" ] || continue; \
	[ -x "$$g" ] || g=$$(command -v "$$g" 2>/dev/null) || continue; \
	"$$g" env GOTOOLCHAIN=local GOFLAGS= go version 2>/dev/null | grep -qE 'go1\.(2[2-9]|[3-9][0-9])' && echo "$$g" && exit 0; \
	done; echo "$(HOST_DIR)/bin/go")

ifeq ($(SUPERVISORD_GO_BIN),$(HOST_DIR)/bin/go)
SUPERVISORD_GO_TOOLCHAIN = go1.22.12
else
SUPERVISORD_GO_TOOLCHAIN = local
endif

SUPERVISORD_GO_ENV = \
	CGO_ENABLED=0 \
	GOTOOLCHAIN=$(SUPERVISORD_GO_TOOLCHAIN) \
	GOFLAGS= \
	GOPROXY=https://proxy.golang.org,direct \
	GOSUMDB=sum.golang.org \
	PATH=$(patsubst %/,%,$(dir $(SUPERVISORD_GO_BIN))):$(PATH)

SUPERVISORD_SHIM_SRCDIR = $(SUPERVISORD_PKGDIR)/supervisorctl-shim

define SUPERVISORD_BUILD_CMDS
	mkdir -p $(@D)/bin
	cd $(@D); \
	$(SUPERVISORD_GO_ENV) $(SUPERVISORD_GO_BIN) run github.com/UnnoTed/fileb0x@v1.1.4 b0x.yaml
	cd $(@D); \
	$(SUPERVISORD_GO_ENV) \
		GOOS=linux GOARCH=arm64 \
		$(SUPERVISORD_GO_BIN) build -tags release -ldflags="-s -w" \
			-o $(@D)/bin/supervisord .
	cd $(SUPERVISORD_SHIM_SRCDIR); \
	$(SUPERVISORD_GO_ENV) \
		GOOS=linux GOARCH=arm64 \
		$(SUPERVISORD_GO_BIN) build -ldflags="-s -w" \
			-o $(@D)/bin/supervisorctl .
endef

define SUPERVISORD_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/supervisord \
		$(TARGET_DIR)/usr/bin/supervisord
	$(INSTALL) -D -m 0755 $(@D)/bin/supervisorctl \
		$(TARGET_DIR)/usr/bin/supervisorctl
endef

$(eval $(generic-package))
