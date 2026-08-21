#!/usr/bin/env bash
# Fetch micronet linux/arm64 binary.
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: MICRONET_GITHUB_REPO, MICRONET_ARTIFACT_NAME,
#                MICRONET_CI_WORKFLOW, MICRONET_FILES
# Token: BIGFRED_NATIVE_TOKEN / GH_TOKEN / GITHUB_TOKEN (required for tip refs)
set -euo pipefail

REF="${1:?usage: $0 <ref> <output.tar>}"
OUT="${2:?usage: $0 <ref> <output.tar>}"

command -v go >/dev/null 2>&1 || {
	echo "error: go is required (install Go toolchain)" >&2
	exit 1
}

export GOPROXY="${GOPROXY:-direct}"
FETCH_PKG="${FETCH_PKG:-github.com/dcc-bigfred/common/cmd/fetch@v0.1.3}"

REPO="${MICRONET_GITHUB_REPO:-dcc-bigfred/micronet}"
ARTIFACT="${MICRONET_ARTIFACT_NAME:-binaries-arm64}"
WORKFLOW="${MICRONET_CI_WORKFLOW:-ci.yml}"
FILES="${MICRONET_FILES:-micronet-linux-arm64:bin/micronet}"

exec go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
