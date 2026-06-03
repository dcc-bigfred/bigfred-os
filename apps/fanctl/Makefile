# Per-app targets; aggregate build: make -C .. build  →  apps/.bin/fanctl

ROOT := $(abspath ../..)

.PHONY: test

test:
	cd "$(ROOT)" && go test ./apps/fanctl/...
