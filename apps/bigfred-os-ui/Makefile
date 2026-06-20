ROOT     := $(abspath ../..)
OUT      ?= $(ROOT)/apps/.bin/bigfred-os-ui
WEB_DIR  := web

GOOS        ?= linux
GOARCH      ?= arm64
CGO_ENABLED ?= 0

USERNAME ?= admin
PASSWORD ?= admin
HTTP_ADDR ?= 0.0.0.0:8090
LOG_ROOTS ?= /data/logs,/var/log

.PHONY: all build web-build test dev-backend dev-web clean

all: build

web-build:
	cd "$(WEB_DIR)" && npm ci && npm run build

build: web-build
	cd "$(ROOT)" && CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags="-s -w" -o "$(OUT)" ./apps/bigfred-os-ui

test:
	cd "$(ROOT)" && go test ./apps/bigfred-os-ui/...

dev-backend: web-build
	cd "$(ROOT)" && go run ./apps/bigfred-os-ui \
		--http "$(HTTP_ADDR)" \
		--username "$(USERNAME)" \
		--password "$(PASSWORD)" \
		--log-roots "$(LOG_ROOTS)" \
		--static-dir apps/bigfred-os-ui/web/dist

dev-web:
	cd "$(WEB_DIR)" && npm run dev

clean:
	rm -rf "$(WEB_DIR)/node_modules" "$(WEB_DIR)/dist"
	rm -f "$(OUT)"
