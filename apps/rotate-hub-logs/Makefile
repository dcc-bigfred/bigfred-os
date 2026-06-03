# Per-app targets; aggregate build: make -C .. build  →  apps/.bin/rotate-hub-logs

ROOT := $(abspath ../..)

.PHONY: test

test:
	cd "$(ROOT)" && go test ./apps/rotate-hub-logs/...
