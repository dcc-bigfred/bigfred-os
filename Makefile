# BigFred OS — top-level build entrypoints

.PHONY: image image-using-docker docker-image check-docker-rpath relocate-br-host check kernel-status

relocate-br-host:
	@bash "$(REPO_ROOT)/scripts/relocate-br-host.sh"

REPO_ROOT    := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
DOCKER_IMAGE ?= bigfred-hub-os-build
DOCKER_DIR   := $(abspath docker)
# Match host ownership of os/output/ (override: make image-using-docker DOCKER_UID=$(id -u))
DOCKER_UID   ?= 1000
DOCKER_GID   ?= 1000

# Defaults from VERSIONS; CLI/env still override (make BIGFRED_REF=… / env).
include $(REPO_ROOT)/VERSIONS

check:
	bash "$(REPO_ROOT)/scripts/check-os.sh"

kernel-status:
	$(MAKE) -C os kernel-status

image:
	$(MAKE) -C os image \
		BIGFRED_REF=$(BIGFRED_REF) \
		MICROINIT_REF=$(MICROINIT_REF) \
		MICRONET_REF=$(MICRONET_REF) \
		MICRODNS_REF=$(MICRODNS_REF) \
		WIRELESS_PROGRAMMER_REF=$(WIRELESS_PROGRAMMER_REF) \
		BIGFRED_WIZARD_REF=$(BIGFRED_WIZARD_REF) \
		MICROWAF_REF=$(MICROWAF_REF)

docker-image:
	docker build -t $(DOCKER_IMAGE) -f $(DOCKER_DIR)/Dockerfile $(REPO_ROOT)

# Fail only when host tools embed a stale absolute HOST_DIR (host vs Docker path).
# $ORIGIN/../lib is valid and portable — do not treat it as an error.
check-docker-rpath:
	@if [ -f "$(REPO_ROOT)/os/output/host/bin/pkgconf" ]; then \
		expected="$(REPO_ROOT)/os/output/host/lib"; \
		rpath=$$(readelf -d "$(REPO_ROOT)/os/output/host/bin/pkgconf" 2>/dev/null | \
			sed -n 's/.*\(RUN\)\?PATH.*\[\(.*\)\].*/\2/p' | head -1); \
		if [ -n "$$rpath" ] && [ "$$rpath" != "$$expected" ] && [ "$$rpath" != '$$ORIGIN/../lib' ]; then \
			case "$$rpath" in \
			/*) \
				echo "error: os/output/host RUNPATH=$$rpath"; \
				echo "       expected $$expected or \$$ORIGIN/../lib (stale absolute path)."; \
				echo "Fix: rm -rf os/output && make image-using-docker"; \
				exit 1 ;; \
			esac; \
		fi; \
	fi

image-using-docker: docker-image check-docker-rpath relocate-br-host
	docker run --rm \
		-u $(DOCKER_UID):$(DOCKER_GID) \
		-v "$(REPO_ROOT):$(REPO_ROOT)" \
		-w "$(REPO_ROOT)" \
		-e HOME="$(REPO_ROOT)" \
		-e RUSTUP_HOME=/usr/local/rustup \
		-e CARGO_HOME=/usr/local/cargo \
		-e MAKEFLAGS="-j$$(nproc 2>/dev/null || echo 4)" \
		-e BIGFRED_REF="$(BIGFRED_REF)" \
		-e MICROINIT_REF="$(MICROINIT_REF)" \
		-e MICRONET_REF="$(MICRONET_REF)" \
		-e MICRODNS_REF="$(MICRODNS_REF)" \
		-e WIRELESS_PROGRAMMER_REF="$(WIRELESS_PROGRAMMER_REF)" \
		-e BIGFRED_WIZARD_REF="$(BIGFRED_WIZARD_REF)" \
		-e MICROWAF_REF="$(MICROWAF_REF)" \
		-e GITHUB_TOKEN \
		-e GH_TOKEN \
		-e BIGFRED_NATIVE_TOKEN \
		-e GITHUB_ACTOR \
		$(DOCKER_IMAGE) \
		make image \
			BIGFRED_REF=$(BIGFRED_REF) \
			MICROINIT_REF=$(MICROINIT_REF) \
			MICRONET_REF=$(MICRONET_REF) \
			MICRODNS_REF=$(MICRODNS_REF) \
			WIRELESS_PROGRAMMER_REF=$(WIRELESS_PROGRAMMER_REF) \
			BIGFRED_WIZARD_REF=$(BIGFRED_WIZARD_REF) \
			MICROWAF_REF=$(MICROWAF_REF)
