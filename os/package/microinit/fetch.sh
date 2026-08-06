#!/usr/bin/env bash
# Fetch microinit linux/arm64 binaries (microinit + optional shutdown).
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

export GITHUB_REPO="${MICROINIT_GITHUB_REPO:-dcc-bigfred/microinit}"
export ARTIFACT_NAME="${MICROINIT_ARTIFACT_NAME:-binaries-arm64}"
export CI_WORKFLOW="${MICROINIT_CI_WORKFLOW:-ci.yml}"

if [[ -n "${MICROINIT_FILES:-}" ]]; then
	export FILES="${MICROINIT_FILES}"
	exec "${FETCH}" "${REF}" "${OUT}"
fi

# Prefer microinit + shutdown; fall back to microinit only.
export FILES="microinit-linux-arm64:bin/microinit,shutdown-linux-arm64:bin/shutdown"
if "${FETCH}" "${REF}" "${OUT}"; then
	exit 0
fi
echo "warning: shutdown layer missing; fetching microinit only" >&2
export FILES="microinit-linux-arm64:bin/microinit"
exec "${FETCH}" "${REF}" "${OUT}"
