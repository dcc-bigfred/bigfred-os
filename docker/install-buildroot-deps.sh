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
	git \
	libncurses-dev \
	libssl-dev \
	debianutils \
	flex \
	libfl2 \
	bison \
	nodejs \
	npm

# ORAS — pull BigFred hub OCI bundle (package/bigfred)
ORAS_VERSION="${ORAS_VERSION:-1.2.2}"
ORAS_ARCH="$(uname -m)"
case "${ORAS_ARCH}" in
	x86_64|amd64) ORAS_ARCH=amd64 ;;
	aarch64|arm64) ORAS_ARCH=arm64 ;;
	*)
		echo "error: unsupported arch for oras: ${ORAS_ARCH}" >&2
		exit 1
		;;
esac
tmp_oras="$(mktemp -d)"
trap 'rm -rf "${tmp_oras}"' EXIT
curl -fsSL \
	"https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_${ORAS_ARCH}.tar.gz" \
	| tar -xz -C "${tmp_oras}"
install -m 0755 "${tmp_oras}/oras" /usr/local/bin/oras
oras version
