#!/usr/bin/env bash
# Fetch microdns linux/arm64 binary.
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: MICRODNS_GITHUB_REPO, MICRODNS_ARTIFACT_NAME,
#                MICRODNS_CI_WORKFLOW, MICRODNS_FILES
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

REPO="${MICRODNS_GITHUB_REPO:-dcc-bigfred/microdns}"
ARTIFACT="${MICRODNS_ARTIFACT_NAME:-binaries-arm64}"
WORKFLOW="${MICRODNS_CI_WORKFLOW:-ci.yml}"
FILES="${MICRODNS_FILES:-microdns-linux-arm64:bin/microdns}"

exec go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
