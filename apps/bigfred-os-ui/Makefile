ROOT     := $(abspath ../..)
BIN_DIR  ?= $(ROOT)/apps/.bin
BIN_BASE := bigfred-os-ui
WEB_DIR  := web

GOOS        ?= linux
GOARCH      ?= arm64
CGO_ENABLED ?= 1

HOST_OS   := $(shell go env GOHOSTOS)
HOST_ARCH := $(shell go env GOHOSTARCH)

OUT_HOST   := $(BIN_DIR)/$(BIN_BASE)-$(HOST_OS)-$(HOST_ARCH)
OUT_TARGET := $(BIN_DIR)/$(BIN_BASE)-$(GOOS)-$(GOARCH)

HOST_DIR    ?= $(ROOT)/os/output/host
STAGING_DIR ?= $(ROOT)/os/output/staging
HOST_CC     := $(HOST_DIR)/bin/aarch64-buildroot-linux-musl-gcc

HTTP_ADDR ?= 0.0.0.0:8090
LOG_ROOTS ?= /data/logs,/var/log
USERNAME  ?= root
PASSWORD  ?= root

TARGET_BUILD_TAGS := pam
HOST_BUILD_TAGS   := !pam
LDFLAGS           := -ldflags="-s -w"
TARGET_GO_FLAGS   := -tags $(TARGET_BUILD_TAGS) $(LDFLAGS)
HOST_GO_FLAGS     := -tags $(HOST_BUILD_TAGS) $(LDFLAGS)

# Cross-compile flags — only for the hub target binary.
CROSS_CC :=
CROSS_CGO_CFLAGS :=
CROSS_CGO_LDFLAGS :=
ifeq ($(CGO_ENABLED),1)
ifneq ($(wildcard $(HOST_CC)),)
  CROSS_CC := $(HOST_CC)
  CROSS_CGO_CFLAGS := --sysroot=$(STAGING_DIR)
  CROSS_CGO_LDFLAGS := --sysroot=$(STAGING_DIR) -L$(STAGING_DIR)/usr/lib -Wl,-rpath-link,$(STAGING_DIR)/usr/lib -lpam
else
  ifeq ($(GOOS)/$(GOARCH),linux/arm64)
    $(warning bigfred-os-ui: Buildroot toolchain not found at $(HOST_CC))
    $(warning Run: make -C os defconfig && make -C os/output linux-pam)
  endif
  CROSS_CGO_LDFLAGS := -lpam
endif
endif

.PHONY: all build web-build build-host build-target test dev-backend dev-web clean

all: build

web-build:
	cd "$(WEB_DIR)" && npm ci && npm run build

ifeq ($(HOST_OS)/$(HOST_ARCH),$(GOOS)/$(GOARCH))
build: web-build build-target
else
build: web-build build-host build-target
endif

build-host:
	@mkdir -p "$(BIN_DIR)"
	@echo "==> $(BIN_BASE) ($(HOST_OS)/$(HOST_ARCH))"
	cd "$(ROOT)" && CGO_ENABLED=0 GOOS=$(HOST_OS) GOARCH=$(HOST_ARCH) CC= \
		go build $(HOST_GO_FLAGS) -o "$(OUT_HOST)" ./apps/bigfred-os-ui

build-target:
	@mkdir -p "$(BIN_DIR)"
	@echo "==> $(BIN_BASE) ($(GOOS)/$(GOARCH))"
	cd "$(ROOT)" && CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		CC="$(CROSS_CC)" CGO_CFLAGS="$(CROSS_CGO_CFLAGS)" CGO_LDFLAGS="$(CROSS_CGO_LDFLAGS)" \
		go build $(TARGET_GO_FLAGS) -o "$(OUT_TARGET)" ./apps/bigfred-os-ui

test:
	cd "$(ROOT)" && CGO_ENABLED=0 go test ./apps/bigfred-os-ui/...

dev-backend: web-build
	cd "$(ROOT)" && CGO_ENABLED=0 GOOS= GOARCH= CC= \
		go run -tags '!pam' ./apps/bigfred-os-ui \
		--http "$(HTTP_ADDR)" \
		--username "$(USERNAME)" \
		--password "$(PASSWORD)" \
		--log-roots "$(LOG_ROOTS)" \
		--static-dir apps/bigfred-os-ui/web/dist

dev-web:
	cd "$(WEB_DIR)" && npm run dev

clean:
	rm -rf "$(WEB_DIR)/node_modules" "$(WEB_DIR)/dist"
	rm -f "$(OUT_HOST)" "$(OUT_TARGET)"
