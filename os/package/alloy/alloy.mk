################################################################################
#
# Grafana Alloy — built as a pure-Go static binary for the musl rootfs.
#
# Official GitHub release zips (alloy-linux-arm64.zip) are dynamically linked
# against glibc (interpreter /lib/ld-linux-aarch64.so.1). On musl that surfaces
# as "alloy: not found" even though the file exists. Building with CGO disabled
# produces a static binary that runs on musl.
#
################################################################################

ALLOY_VERSION = 1.17.1
ALLOY_SITE = $(call github,grafana,alloy,v$(ALLOY_VERSION))
ALLOY_LICENSE = Apache-2.0
ALLOY_LICENSE_FILES = LICENSE

ifeq ($(BR2_aarch64),y)
ALLOY_GOARCH = arm64
else ifeq ($(BR2_arm)$(BR2_armeb),y)
ALLOY_GOARCH = arm
ALLOY_GOARM = 7
else ifeq ($(BR2_x86_64),y)
ALLOY_GOARCH = amd64
else
$(error alloy: unsupported target architecture $(BR2_ARCH))
endif

# Prefer Go from PATH (docker/install-go.sh → /usr/local/go). Buildroot's
# host-go is older than Alloy's go.mod (1.26+) and pins GOTOOLCHAIN=local.
# Resolve at recipe time so PATH from the docker/image wrapper is visible.
ALLOY_GO = go

# No UI embed (needs npm generate-ui) and no systemd journal (needs CGO).
# netgo/osusergo keep the binary free of libc DNS/user lookups.
define ALLOY_BUILD_CMDS
	@command -v $(ALLOY_GO) >/dev/null 2>&1 || { \
		echo "error: alloy: go not found on PATH — install via docker/install-go.sh (Go ≥ 1.26)" >&2; \
		exit 1; \
	}
	mkdir -p $(@D)/bin
	cd $(@D) && \
		CGO_ENABLED=0 \
		GOOS=linux \
		GOARCH=$(ALLOY_GOARCH) \
		$(if $(ALLOY_GOARM),GOARM=$(ALLOY_GOARM)) \
		GOTOOLCHAIN=auto \
		GOPROXY=https://proxy.golang.org,direct \
		$(ALLOY_GO) build -trimpath -buildvcs=false \
			-tags 'netgo osusergo' \
			-ldflags '-s -w' \
			-o $(@D)/bin/alloy \
			.
endef

define ALLOY_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/bin/alloy $(TARGET_DIR)/usr/bin/alloy
	# Fail the image build if we somehow produced a glibc-dynamic binary.
	if readelf -l $(TARGET_DIR)/usr/bin/alloy 2>/dev/null \
		| grep -q 'Requesting program interpreter: /lib/ld-linux'; then \
		echo "error: alloy is dynamically linked against glibc" >&2; \
		file $(TARGET_DIR)/usr/bin/alloy >&2; \
		exit 1; \
	fi
endef

$(eval $(generic-package))
