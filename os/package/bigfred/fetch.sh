#!/usr/bin/env bash
# Fetch bigfred hub linux/arm64 binaries (loco-server + remote-icmp).
# Usage: fetch.sh <ref> <output.tar>
#
# Env overrides: BIGFRED_GITHUB_REPO, BIGFRED_ARTIFACT_NAME,
#                BIGFRED_CI_WORKFLOW, BIGFRED_FILES
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

REPO="${BIGFRED_GITHUB_REPO:-dcc-bigfred/bigfred}"
ARTIFACT="${BIGFRED_ARTIFACT_NAME:-binaries}"
WORKFLOW="${BIGFRED_CI_WORKFLOW:-ci.yml}"
FILES="${BIGFRED_FILES:-loco-server-linux-arm64:bin/bigfred,bigfred-remote-icmp-linux-arm64:bin/bigfred-remote-icmp}"

set +e
go run "${FETCH_PKG}" \
	--repo="${REPO}" \
	--files="${FILES}" \
	--artifact="${ARTIFACT}" \
	--workflow="${WORKFLOW}" \
	"${REF}" "${OUT}"
rc=$?
set -e

if [[ "${rc}" -eq 0 ]]; then
	exit 0
fi

if [[ "${REF}" == "latest-release" ]]; then
	echo "warning: latest-release missing; trying master" >&2
	exec go run "${FETCH_PKG}" \
		--repo="${REPO}" \
		--files="${FILES}" \
		--artifact="${ARTIFACT}" \
		--workflow="${WORKFLOW}" \
		master "${OUT}"
fi
exit "${rc}"
