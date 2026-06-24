# Build all hub apps into apps/.bin/<name> (linux/arm64, static)

ROOT        := $(abspath ..)
APPS_DIR    := $(CURDIR)
BIN_DIR     := $(APPS_DIR)/.bin

GOOS        ?= linux
GOARCH      ?= arm64
CGO_ENABLED ?= 0

# Subdirectories of apps/ that contain main.go
APPS        := $(patsubst %/main.go,%,$(wildcard */main.go))
GENERIC_APPS := $(filter-out bigfred-os-ui,$(APPS))
BINARIES    := $(addprefix $(BIN_DIR)/,$(APPS))

GO_BUILD    = cd "$(ROOT)" && CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags="-s -w" -o

.PHONY: all build test clean $(APPS)

all: build

build: $(BINARIES)

$(BIN_DIR)/bigfred-os-ui: bigfred-os-ui/main.go
	@$(MAKE) -C bigfred-os-ui build GOOS=$(GOOS) GOARCH=$(GOARCH)

$(BIN_DIR)/%: %/main.go
	@if [ "$*" = "bigfred-os-ui" ]; then exit 0; fi
	@mkdir -p "$(BIN_DIR)"
	@echo "==> apps/$*"
	$(GO_BUILD) "$(BIN_DIR)/$*" ./apps/$*

test:
	cd "$(ROOT)" && go test $(addprefix ./apps/,$(addsuffix /...,$(APPS)))

clean:
	rm -rf "$(BIN_DIR)"

# Build a single app: make rotate-hub-logs
$(APPS):
	@$(MAKE) --no-print-directory $(BIN_DIR)/$@
