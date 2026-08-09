#!/bin/sh
# Post-build runner: executes post-build.d/*.sh in lexical order.
# Buildroot invokes this as BR2_ROOTFS_POST_BUILD_SCRIPT (TARGET_DIR set).

set -e

HUB="${BR2_EXTERNAL_BIGFRED_HUB_PATH:-$(dirname "$0")/../..}"
REPO_ROOT="$(cd "${HUB}/.." && pwd)"

# Shared commit identity (consumed by post-build.d/60-identity.sh).
COMMIT="unknown"
if command -v git >/dev/null 2>&1 && [ -d "${REPO_ROOT}/.git" ]; then
	COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || true)"
	[ -n "${COMMIT}" ] || COMMIT="unknown"
elif [ -n "${GITHUB_SHA:-}" ]; then
	COMMIT="${GITHUB_SHA}"
fi
export HUB REPO_ROOT COMMIT

D="${HUB}/board/bigfred_hub/post-build.d"
for script in "${D}"/*.sh; do
	[ -f "${script}" ] || continue
	echo "post-build: $(basename "${script}")"
	# shellcheck source=/dev/null
	. "${script}"
done

exit 0
