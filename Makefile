# BigFred OS — top-level build entrypoints

.PHONY: image image-using-docker docker-image check-docker-rpath

REPO_ROOT    := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
DOCKER_IMAGE ?= bigfred-hub-os-build
DOCKER_DIR   := $(abspath docker)
# Match host ownership of os/output/ (override: make image-using-docker DOCKER_UID=$(id -u))
DOCKER_UID   ?= 1000
DOCKER_GID   ?= 1000

image:
	$(MAKE) -C os image

docker-image:
	docker build -t $(DOCKER_IMAGE) -f $(DOCKER_DIR)/Dockerfile $(DOCKER_DIR)

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

image-using-docker: docker-image check-docker-rpath
	docker run --rm \
		-u $(DOCKER_UID):$(DOCKER_GID) \
		-v "$(REPO_ROOT):$(REPO_ROOT)" \
		-w "$(REPO_ROOT)" \
		-e HOME="$(REPO_ROOT)" \
		-e MAKEFLAGS="-j$$(nproc 2>/dev/null || echo 4)" \
		$(DOCKER_IMAGE) \
		make image
