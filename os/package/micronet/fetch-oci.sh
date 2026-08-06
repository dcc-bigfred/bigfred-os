#!/usr/bin/env bash
# Pull micronet linux/arm64 OCI bundle from GHCR and write a tar for Buildroot.
# Usage: fetch-oci.sh <oci-tag> <output.tar>
#
# Env:
#   MICRONET_OCI_IMAGE — default ghcr.io/dcc-bigfred/micronet-linux-arm64
#   GITHUB_TOKEN / GH_TOKEN / BIGFRED_NATIVE_TOKEN — optional GHCR auth
set -euo pipefail

TAG="${1:?usage: $0 <oci-tag> <output.tar>}"
OUT="${2:?usage: $0 <oci-tag> <output.tar>}"
IMAGE="${MICRONET_OCI_IMAGE:-ghcr.io/dcc-bigfred/micronet-linux-arm64}"

if ! command -v oras >/dev/null 2>&1; then
	echo "error: oras not on PATH (install via docker/install-buildroot-deps.sh)" >&2
	exit 1
fi

TOKEN="${BIGFRED_NATIVE_TOKEN:-${GH_TOKEN:-${GITHUB_TOKEN:-}}}"
if [[ -n "${TOKEN}" ]]; then
	USER="${GITHUB_ACTOR:-oauth2}"
	echo "${TOKEN}" | oras login ghcr.io -u "${USER}" --password-stdin >/dev/null
fi

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "${tmpdir}"; }
trap cleanup EXIT

pull_tag() {
	local t="$1"
	rm -rf "${tmpdir:?}"/*
	mkdir -p "${tmpdir}"
	echo "Pulling ${IMAGE}:${t}…"
	oras pull "${IMAGE}:${t}" -o "${tmpdir}"
}

if ! pull_tag "${TAG}"; then
	echo "error: could not pull ${IMAGE}:${TAG}" >&2
	exit 1
fi

DHCP="${tmpdir}/configure-dhcp-linux-arm64"
ETH="${tmpdir}/configure-ethernet-linux-arm64"
if [[ ! -f "${DHCP}" ]] || [[ ! -f "${ETH}" ]]; then
	echo "error: expected configure-dhcp-linux-arm64 and configure-ethernet-linux-arm64, found:" >&2
	find "${tmpdir}" -type f >&2
	exit 1
fi

chmod 755 "${DHCP}" "${ETH}"
mkdir -p "$(dirname "${OUT}")"
staged="${tmpdir}/stage"
mkdir -p "${staged}/bin"
cp -f "${DHCP}" "${staged}/bin/configure-dhcp"
cp -f "${ETH}" "${staged}/bin/configure-ethernet"
tar -C "${staged}" -cf "${OUT}" bin
echo "Wrote ${OUT} ($(wc -c < "${OUT}") bytes) from ${IMAGE}:${TAG}"
