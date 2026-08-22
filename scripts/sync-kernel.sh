#!/usr/bin/env bash
# Compare (or bump) the pinned hub-kernel Release SHA against linux-prebuilt.hash.
# Usage:
#   scripts/sync-kernel.sh              # status of the pinned version
#   scripts/sync-kernel.sh v6.18.0-r2   # status of another tag
#   scripts/sync-kernel.sh --bump [v*]  # rewrite .mk version + .hash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MK="$ROOT/os/package/linux-prebuilt/linux-prebuilt.mk"
HASH="$ROOT/os/package/linux-prebuilt/linux-prebuilt.hash"
REPO="${LINUX_PREBUILT_GITHUB_REPO:-dcc-bigfred/hub-kernel}"

bump=0
tag=""
for arg in "$@"; do
	case "$arg" in
	--bump) bump=1 ;;
	-h|--help)
		sed -n '2,8p' "$0"
		exit 0
		;;
	v*) tag="$arg" ;;
	*)
		echo "usage: $0 [--bump] [v6.18.0-rN]" >&2
		exit 2
		;;
	esac
done

pinned="$(sed -n 's/^LINUX_PREBUILT_VERSION = //p' "$MK" | head -1)"
if [[ -z "$pinned" ]]; then
	echo "error: LINUX_PREBUILT_VERSION missing in $MK" >&2
	exit 1
fi
if [[ -z "$tag" ]]; then
	tag="v${pinned}"
fi
version="${tag#v}"
asset="bigfred-kernel-rpi5-${tag}.tar.xz"

remote="$(curl -fsSL "https://github.com/${REPO}/releases/download/${tag}/${asset}.sha256")"
remote_sha="$(awk '{ print $1 }' <<<"$remote")"
if [[ ! "$remote_sha" =~ ^[0-9a-f]{64}$ ]]; then
	echo "error: could not parse SHA-256 for ${tag}" >&2
	echo "$remote" >&2
	exit 1
fi

local_sha=""
if [[ -f "$HASH" ]]; then
	local_sha="$(awk -v name="$asset" '$1 == "sha256" && $3 == name { print $2; exit }' "$HASH")"
fi

echo "repo:    ${REPO}"
echo "tag:     ${tag}"
echo "asset:   ${asset}"
echo "remote:  ${remote_sha}"
echo "pinned:  ${local_sha:-"(missing)"}  (LINUX_PREBUILT_VERSION=${pinned})"

if [[ "$local_sha" == "$remote_sha" && "$pinned" == "$version" ]]; then
	echo "status:  match (no download needed if dl/ cache has this file)"
	exit 0
fi

if [[ "$bump" -eq 0 ]]; then
	echo "status:  differ (re-run with --bump to pin ${tag})"
	exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
sed "s/^LINUX_PREBUILT_VERSION = .*/LINUX_PREBUILT_VERSION = ${version}/" "$MK" > "$tmp"
mv "$tmp" "$MK"
cat > "$HASH" <<EOF
# From https://github.com/${REPO}/releases/download/${tag}/${asset}.sha256
sha256  ${remote_sha}  ${asset}
EOF
echo "status:  bumped ${MK} and ${HASH} to ${tag}"
