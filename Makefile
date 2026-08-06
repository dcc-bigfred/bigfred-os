# BigFred OS — top-level build entrypoints

.PHONY: image image-using-docker docker-image check-docker-rpath relocate-br-host

relocate-br-host:
	@bash "$(REPO_ROOT)/scripts/relocate-br-host.sh"

REPO_ROOT    := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
DOCKER_IMAGE ?= bigfred-hub-os-build
DOCKER_DIR   := $(abspath docker)
# Match host ownership of os/output/ (override: make image-using-docker DOCKER_UID=$(id -u))
DOCKER_UID   ?= 1000
DOCKER_GID   ?= 1000
# Hub OCI tag: master | latest-release | v* | sha-<7>
BIGFRED_OCI_TAG ?= latest-release
# microinit PID 1 OCI tag: main | sha-<7>
MICROINIT_OCI_TAG ?= main
# microdns mDNS advertiser OCI tag: main | sha-<7>
MICRODNS_OCI_TAG ?= main

image:
	$(MAKE) -C os image BIGFRED_OCI_TAG=$(BIGFRED_OCI_TAG) MICROINIT_OCI_TAG=$(MICROINIT_OCI_TAG) MICRODNS_OCI_TAG=$(MICRODNS_OCI_TAG)

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
		-e BIGFRED_OCI_TAG="$(BIGFRED_OCI_TAG)" \
		-e MICROINIT_OCI_TAG="$(MICROINIT_OCI_TAG)" \
		-e MICRODNS_OCI_TAG="$(MICRODNS_OCI_TAG)" \
		-e GITHUB_TOKEN \
		-e GH_TOKEN \
		-e BIGFRED_NATIVE_TOKEN \
		-e GITHUB_ACTOR \
		$(DOCKER_IMAGE) \
		make image BIGFRED_OCI_TAG=$(BIGFRED_OCI_TAG) MICROINIT_OCI_TAG=$(MICROINIT_OCI_TAG) MICRODNS_OCI_TAG=$(MICRODNS_OCI_TAG)
