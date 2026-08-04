# Go hub app → apps/.bin/configure-ethernet

ROOT    := $(abspath ../..)
BIN_DIR ?= $(ROOT)/apps/.bin
APP     := $(notdir $(CURDIR))

GOOS        ?= linux
GOARCH      ?= arm64
CGO_ENABLED ?= 0

.PHONY: all build test clean

all: build

build:
	@mkdir -p "$(BIN_DIR)"
	@echo "==> apps/$(APP)"
	cd "$(ROOT)" && CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o "$(BIN_DIR)/$(APP)" ./apps/$(APP)

test:
	cd "$(ROOT)" && go test ./apps/$(APP)/...

clean:
	rm -f "$(BIN_DIR)/$(APP)"
