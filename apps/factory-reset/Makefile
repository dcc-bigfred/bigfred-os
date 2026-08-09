# factory-reset — Rust, static aarch64 musl → apps/.bin/factory-reset

ROOT    := $(abspath ../..)
BIN_DIR ?= $(ROOT)/apps/.bin
TARGET  ?= aarch64-unknown-linux-musl

# Prefer Buildroot host cross-gcc when present (hub OS tree).
BR_HOST_GCC := $(ROOT)/os/output/host/bin/aarch64-buildroot-linux-musl-gcc
ifeq ($(wildcard $(BR_HOST_GCC)),)
  LINKER ?= aarch64-linux-musl-gcc
else
  LINKER ?= $(BR_HOST_GCC)
endif

CARGO   ?= cargo
export RUSTUP_TOOLCHAIN ?= stable

.PHONY: all build test clean host

all: build

build:
	@command -v cargo >/dev/null 2>&1 || { \
		echo "error: cargo not found — install Rust (sudo docker/install-rust.sh)" >&2; \
		echo "       or rebuild the Docker image: make docker-image" >&2; \
		exit 127; \
	}
	@mkdir -p "$(BIN_DIR)"
	@echo "==> apps/factory-reset (Rust $(TARGET))"
	CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER="$(LINKER)" \
	RUSTFLAGS='-C target-feature=+crt-static' \
		$(CARGO) build --release --target $(TARGET)
	cp -f "$(or $(CARGO_TARGET_DIR),target)/$(TARGET)/release/factory-reset" "$(BIN_DIR)/factory-reset"
	chmod 755 "$(BIN_DIR)/factory-reset"
	@echo "wrote $(BIN_DIR)/factory-reset"

# Native build + unit tests (no block devices required).
test host:
	$(CARGO) test

clean:
	$(CARGO) clean
	rm -f "$(BIN_DIR)/factory-reset"
