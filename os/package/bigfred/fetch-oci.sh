#!/usr/bin/env bash
# Pull bigfred hub linux/arm64 OCI bundle from GHCR and write a tar for Buildroot.
# Usage: fetch-oci.sh <oci-tag> <output.tar>
#
# Env:
#   BIGFRED_HUB_OCI_IMAGE — default ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64
#   GITHUB_TOKEN / GH_TOKEN / BIGFRED_NATIVE_TOKEN — optional GHCR auth
set -euo pipefail

TAG="${1:?usage: $0 <oci-tag> <output.tar>}"
OUT="${2:?usage: $0 <oci-tag> <output.tar>}"
IMAGE="${BIGFRED_HUB_OCI_IMAGE:-ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64}"

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
	# Before the first v* release after hub OCI landed, :latest-release is absent.
	if [[ "${TAG}" == "latest-release" ]] && pull_tag "master"; then
		echo "warning: ${IMAGE}:latest-release missing; used :master" >&2
	else
		echo "error: could not pull ${IMAGE}:${TAG}" >&2
		exit 1
	fi
fi

SERVER="${tmpdir}/loco-server-linux-arm64"
ICMP="${tmpdir}/bigfred-remote-icmp-linux-arm64"
if [[ ! -f "${SERVER}" || ! -f "${ICMP}" ]]; then
	echo "error: expected loco-server-linux-arm64 and bigfred-remote-icmp-linux-arm64, found:" >&2
	find "${tmpdir}" -type f >&2
	exit 1
fi

chmod 755 "${SERVER}" "${ICMP}"
mkdir -p "$(dirname "${OUT}")"
# Stable layout for EXTRACT/INSTALL: bin/bigfred + bin/bigfred-remote-icmp
staged="${tmpdir}/stage"
mkdir -p "${staged}/bin"
cp -f "${SERVER}" "${staged}/bin/bigfred"
cp -f "${ICMP}" "${staged}/bin/bigfred-remote-icmp"
tar -C "${staged}" -cf "${OUT}" bin
echo "Wrote ${OUT} ($(wc -c < "${OUT}") bytes) from ${IMAGE}:${TAG}"
