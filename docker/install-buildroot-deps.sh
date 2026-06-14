#!/usr/bin/env bash
# Host packages for Buildroot (https://buildroot.org/downloads/manual/manual.html#requirement).
# Used by CI, docker/Dockerfile, and optionally on bare Ubuntu/Debian hosts.

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	exec sudo --preserve-env=DEBIAN_FRONTEND "$0" "$@"
fi

export DEBIAN_FRONTEND="${DEBIAN_FRONTEND:-noninteractive}"

apt-get update
apt-get install -y --no-install-recommends \
	build-essential \
	sed \
	make \
	binutils \
	gcc \
	g++ \
	bash \
	patch \
	gzip \
	bzip2 \
	perl \
	tar \
	cpio \
	python3 \
	unzip \
	rsync \
	wget \
	curl \
	file \
	bc \
	ca-certificates \
	libncurses-dev \
	libssl-dev \
	debianutils \
	flex \
	libfl2 \
	bison
