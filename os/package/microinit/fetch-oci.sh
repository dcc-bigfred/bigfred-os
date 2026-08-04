#!/usr/bin/env bash
# Pull microinit linux/arm64 OCI bundle from GHCR and write a tar for Buildroot.
# Usage: fetch-oci.sh <oci-tag> <output.tar>
#
# Env:
#   MICROINIT_OCI_IMAGE — default ghcr.io/dcc-bigfred/microinit-linux-arm64
#   GITHUB_TOKEN / GH_TOKEN / BIGFRED_NATIVE_TOKEN — optional GHCR auth
set -euo pipefail

TAG="${1:?usage: $0 <oci-tag> <output.tar>}"
OUT="${2:?usage: $0 <oci-tag> <output.tar>}"
IMAGE="${MICROINIT_OCI_IMAGE:-ghcr.io/dcc-bigfred/microinit-linux-arm64}"

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

BIN="${tmpdir}/microinit-linux-arm64"
if [[ ! -f "${BIN}" ]]; then
	echo "error: expected microinit-linux-arm64 in OCI layer, found:" >&2
	find "${tmpdir}" -type f >&2
	exit 1
fi

chmod 755 "${BIN}"
mkdir -p "$(dirname "${OUT}")"
# Stable layout for EXTRACT/INSTALL: bin/microinit (+ optional bin/shutdown)
# Do not install OCI early-boot.sh / unmount.sh — hub overlay ships both under
# /etc/microinit/.
staged="${tmpdir}/stage"
mkdir -p "${staged}/bin"
cp -f "${BIN}" "${staged}/bin/microinit"
if [[ -f "${tmpdir}/shutdown-linux-arm64" ]]; then
	chmod 755 "${tmpdir}/shutdown-linux-arm64"
	cp -f "${tmpdir}/shutdown-linux-arm64" "${staged}/bin/shutdown"
fi
tar -C "${staged}" -cf "${OUT}" bin
echo "Wrote ${OUT} ($(wc -c < "${OUT}") bytes) from ${IMAGE}:${TAG}"
