#!/usr/bin/env bash
# Fetch wireless-programmer linux/arm64 binary.
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: WIRELESS_PROGRAMMER_GITHUB_REPO, WIRELESS_PROGRAMMER_ARTIFACT_NAME,
#                WIRELESS_PROGRAMMER_CI_WORKFLOW, WIRELESS_PROGRAMMER_FILES
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

REPO="${WIRELESS_PROGRAMMER_GITHUB_REPO:-dcc-bigfred/wireless-programmer}"
ARTIFACT="${WIRELESS_PROGRAMMER_ARTIFACT_NAME:-binaries-arm64}"
WORKFLOW="${WIRELESS_PROGRAMMER_CI_WORKFLOW:-ci.yml}"
FILES="${WIRELESS_PROGRAMMER_FILES:-wireless-programmer-linux-arm64:wireless-programmer-linux-arm64}"

exec go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
