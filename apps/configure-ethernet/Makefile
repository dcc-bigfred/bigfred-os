# Per-app targets; aggregate build: make -C .. build  →  apps/.bin/configure-ethernet

ROOT := $(abspath ../..)

.PHONY: test

test:
	cd "$(ROOT)" && go test ./apps/configure-ethernet/...
