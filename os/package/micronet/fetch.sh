#!/usr/bin/env bash
# Fetch micronet linux/arm64 binaries (configure-dhcp + configure-ethernet).
# Usage: fetch.sh <ref> <output.tar>
#
# Env: MICRONET_GITHUB_REPO (default dcc-bigfred/micronet)
#      CI_SCRIPTS_DIR — path to cloned dcc-bigfred/.github (default <repo>/.ci-github)
#      BIGFRED_NATIVE_TOKEN / GH_TOKEN / GITHUB_TOKEN — required for tip refs
set -euo pipefail

REF="${1:?usage: $0 <ref> <output.tar>}"
OUT="${2:?usage: $0 <ref> <output.tar>}"

PKGDIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${PKGDIR}/../../.." && pwd)"
CI_SCRIPTS_DIR="${CI_SCRIPTS_DIR:-${REPO_ROOT}/.ci-github}"
FETCH="${CI_SCRIPTS_DIR}/scripts/fetch-github-binaries.sh"

if [[ ! -x "${FETCH}" ]]; then
	echo "error: missing ${FETCH}" >&2
	echo "hint: from bigfred-os/os run: make ci-scripts  (clones dcc-bigfred/.github @ v2)" >&2
	exit 1
fi

export GITHUB_REPO="${MICRONET_GITHUB_REPO:-dcc-bigfred/micronet}"
export ARTIFACT_NAME="${MICRONET_ARTIFACT_NAME:-binaries-arm64}"
export CI_WORKFLOW="${MICRONET_CI_WORKFLOW:-ci.yml}"
export FILES="${MICRONET_FILES:-configure-dhcp-linux-arm64:bin/configure-dhcp,configure-ethernet-linux-arm64:bin/configure-ethernet}"

exec "${FETCH}" "${REF}" "${OUT}"
