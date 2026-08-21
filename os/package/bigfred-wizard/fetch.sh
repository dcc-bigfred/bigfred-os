#!/usr/bin/env bash
# Fetch bigfred-wizard linux/arm64 binary.
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: BIGFRED_WIZARD_GITHUB_REPO, BIGFRED_WIZARD_ARTIFACT_NAME,
#                BIGFRED_WIZARD_CI_WORKFLOW, BIGFRED_WIZARD_FILES
# Token: BIGFRED_NATIVE_TOKEN / GH_TOKEN / GITHUB_TOKEN (required for tip refs)
set -euo pipefail

REF="${1:?usage: $0 <ref> <output.tar>}"
OUT="${2:?usage: $0 <ref> <output.tar>}"

command -v go >/dev/null 2>&1 || {
	echo "error: go is required (install Go toolchain)" >&2
	exit 1
}

export GOPROXY="${GOPROXY:-direct}"
FETCH_PKG="${FETCH_PKG:-github.com/dcc-bigfred/common/cmd/fetch@v0.1.4}"

REPO="${BIGFRED_WIZARD_GITHUB_REPO:-dcc-bigfred/bigfred-wizard}"
ARTIFACT="${BIGFRED_WIZARD_ARTIFACT_NAME:-binaries-arm64}"
WORKFLOW="${BIGFRED_WIZARD_CI_WORKFLOW:-ci.yml}"
FILES="${BIGFRED_WIZARD_FILES:-bigfred-wizard-linux-arm64:bigfred-wizard-linux-arm64}"

exec go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
