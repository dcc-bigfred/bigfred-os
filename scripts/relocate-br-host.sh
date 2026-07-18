#!/usr/bin/env bash
# Fix Buildroot host/ paths when os/output was built or cached under another directory.
# host-fakeroot embeds FAKEROOT_PREFIX at install time; moving the repo breaks rootfs.ext2.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOST_DIR="${ROOT}/os/output/host"
FAKEROOT="${HOST_DIR}/bin/fakeroot"

if [[ ! -f "${FAKEROOT}" ]]; then
	exit 0
fi

# Already patched to resolve paths from $0.
if grep -q 'bigfred-os: relocatable host paths' "${FAKEROOT}"; then
	exit 0
fi

embedded="$(sed -n 's/^FAKEROOT_PREFIX=\(.*\)/\1/p' "${FAKEROOT}" | head -1)"
if [[ -z "${embedded}" ]]; then
	echo "relocate-br-host: unexpected fakeroot script layout in ${FAKEROOT}" >&2
	exit 1
fi

host_dir="$(cd "${HOST_DIR}" && pwd)"
if [[ "${embedded}" == "${host_dir}" ]]; then
	exit 0
fi

echo "relocate-br-host: ${embedded} -> ${host_dir}"

# Text files only (-I); updates fakeroot, *.pc, wrapper scripts, etc.
mapfile -t files < <(grep -rIlF "${embedded}" "${HOST_DIR}" || true)
if ((${#files[@]} == 0)); then
	echo "relocate-br-host: no references to old prefix found" >&2
	exit 1
fi

escaped_old="${embedded//\\/\\\\}"
escaped_old="${escaped_old//|/\\|}"
escaped_old="${escaped_old//&/\\&}"

for f in "${files[@]}"; do
	sed -i "s|${escaped_old}|${host_dir}|g" "${f}"
done

# Make fakeroot relocatable if the tree moves again (e.g. CI cache vs local checkout).
if ! grep -q 'bigfred-os: relocatable host paths' "${FAKEROOT}"; then
	sed -i \
		-e 's|^FAKEROOT_PREFIX=.*|# bigfred-os: relocatable host paths\nFAKEROOT_BINDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" \&\& pwd)"\nFAKEROOT_PREFIX="$(cd "${FAKEROOT_BINDIR}/.." \&\& pwd)"|' \
		-e 's|^FAKEROOT_BINDIR=.*||' \
		-e "s|^PATHS=.*|PATHS=\${FAKEROOT_PREFIX}/lib:\${FAKEROOT_PREFIX}/lib64/libfakeroot:\${FAKEROOT_PREFIX}/lib32/libfakeroot|" \
		-e 's|^FAKED=.*|FAKED=${FAKEROOT_BINDIR}/faked|' \
		"${FAKEROOT}"
fi

echo "relocate-br-host: updated ${#files[@]} file(s)"
