#!/usr/bin/env bash
# Fetch microinit linux/arm64 binaries (microinit + shutdown).
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: MICROINIT_GITHUB_REPO, MICROINIT_ARTIFACT_NAME,
#                MICROINIT_CI_WORKFLOW, MICROINIT_FILES
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

REPO="${MICROINIT_GITHUB_REPO:-dcc-bigfred/microinit}"
ARTIFACT="${MICROINIT_ARTIFACT_NAME:-binaries-arm64}"
WORKFLOW="${MICROINIT_CI_WORKFLOW:-ci.yml}"
FILES="${MICROINIT_FILES:-microinit-linux-arm64:bin/microinit,shutdown-linux-arm64:bin/shutdown}"

exec go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
