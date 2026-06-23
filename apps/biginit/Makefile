# Per-app targets; aggregate build: make -C .. build  →  apps/.bin/biginit

ROOT := $(abspath ../..)

.PHONY: test

test:
	cd "$(ROOT)" && go test ./apps/biginit/...
