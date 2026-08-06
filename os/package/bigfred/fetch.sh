#!/usr/bin/env bash
# Fetch bigfred hub linux/arm64 binaries (loco-server + remote-icmp).
# Usage: fetch.sh <ref> <output.tar>
set -euo pipefail

REF="${1:?usage: $0 <ref> <output.tar>}"
OUT="${2:?usage: $0 <ref> <output.tar>}"

PKGDIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${PKGDIR}/../../.." && pwd)"
CI_SCRIPTS_DIR="${CI_SCRIPTS_DIR:-${REPO_ROOT}/.ci-github}"
FETCH="${CI_SCRIPTS_DIR}/scripts/fetch-github-binaries.sh"

if [[ ! -x "${FETCH}" ]]; then
	echo "error: missing ${FETCH}" >&2
	echo "hint: from bigfred-os/os run: make ci-scripts" >&2
	exit 1
fi

export GITHUB_REPO="${BIGFRED_GITHUB_REPO:-dcc-bigfred/bigfred}"
export ARTIFACT_NAME="${BIGFRED_ARTIFACT_NAME:-binaries}"
export CI_WORKFLOW="${BIGFRED_CI_WORKFLOW:-ci.yml}"
export FILES="${BIGFRED_FILES:-loco-server-linux-arm64:bin/bigfred,bigfred-remote-icmp-linux-arm64:bin/bigfred-remote-icmp}"

if "${FETCH}" "${REF}" "${OUT}"; then
	exit 0
fi

if [[ "${REF}" == "latest-release" ]]; then
	echo "warning: latest-release missing; trying master" >&2
	exec "${FETCH}" master "${OUT}"
fi
exit 1
