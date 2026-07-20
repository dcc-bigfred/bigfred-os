#!/usr/bin/env bash
# Install the Go toolchain required by go.mod (used by docker/Dockerfile and optional on hosts).
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	exec sudo --preserve-env=DEBIAN_FRONTEND,GO_MOD,GO_VERSION "$0" "$@"
fi

resolve_go_mod() {
	if [ -n "${GO_MOD:-}" ] && [ -f "$GO_MOD" ]; then
		printf '%s\n' "$GO_MOD"
		return 0
	fi
	local script_dir
	script_dir="$(cd "$(dirname "$0")" && pwd)"
	if [ -f "${script_dir}/../go.mod" ]; then
		printf '%s\n' "${script_dir}/../go.mod"
		return 0
	fi
	echo "error: go.mod not found (set GO_MOD to the module file path)" >&2
	return 1
}

GO_MOD_FILE="$(resolve_go_mod)"
GO_MINOR="$(sed -n 's/^go //p' "$GO_MOD_FILE" | head -1 | tr -d ' \r')"
if [ -z "$GO_MINOR" ]; then
	echo "error: could not read Go version from $GO_MOD_FILE" >&2
	exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
x86_64) GOARCH=amd64 ;;
aarch64|arm64) GOARCH=arm64 ;;
*) echo "error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

resolve_latest_patch() {
	local minor="$1"
	local goarch="$2"
	local patch candidate code
	# go.dev JSON only lists the latest two stable releases; probe patch versions directly.
	for patch in $(seq 30 -1 0); do
		candidate="${minor}.${patch}"
		code="$(curl -fsSIL -o /dev/null -w '%{http_code}' "https://go.dev/dl/go${candidate}.linux-${goarch}.tar.gz" 2>/dev/null || true)"
		case "$code" in
		200|302)
			printf '%s\n' "$candidate"
			return 0
			;;
		esac
	done
	echo "error: no Go release found for ${minor}.x on go.dev" >&2
	return 1
}

if [ -n "${GO_VERSION:-}" ]; then
	VERSION="$GO_VERSION"
elif [[ "$GO_MINOR" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	VERSION="$GO_MINOR"
else
	VERSION="$(resolve_latest_patch "$GO_MINOR" "$GOARCH")"
fi

TARBALL="go${VERSION}.linux-${GOARCH}.tar.gz"
URL="https://go.dev/dl/${TARBALL}"

echo "Installing Go ${VERSION} (${URL}) ..."
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
curl -fsSL "$URL" -o "${tmpdir}/${TARBALL}"
rm -rf /usr/local/go
tar -C /usr/local -xzf "${tmpdir}/${TARBALL}"
ln -sf /usr/local/go/bin/go /usr/local/bin/go
ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

/usr/local/go/bin/go version
