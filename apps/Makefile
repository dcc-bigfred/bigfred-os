# Build all hub apps → apps/.bin/
# Language-agnostic: each apps/<name>/Makefile owns `build` / `test` / `clean`.

APPS_DIR := $(CURDIR)
BIN_DIR  := $(APPS_DIR)/.bin

# Every immediate subdirectory that has its own Makefile.
APPS := $(patsubst %/Makefile,%,$(wildcard */Makefile))

.PHONY: all build test clean $(APPS)

all: build

build: $(APPS)

# `make rotate-hub-logs` / `make prepare-nvme` / …
$(APPS):
	@$(MAKE) -C $@ build BIN_DIR="$(BIN_DIR)"

test:
	@for app in $(APPS); do \
		$(MAKE) -C $$app test || exit 1; \
	done

clean:
	@for app in $(APPS); do \
		$(MAKE) -C $$app clean BIN_DIR="$(BIN_DIR)" || true; \
	done
	rm -rf "$(BIN_DIR)"
