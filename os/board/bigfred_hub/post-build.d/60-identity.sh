# Build-time OS identity: git commit, marker dir, /etc/lsb-release, /etc/os-release.
# prepare-nvme / factory-reset refuse to run unless /var/lib/bigfred exists
# (and factory-reset also checks DISTRIB_ID=bigfred-os in /etc/lsb-release).

mkdir -p "${TARGET_DIR}/usr/lib/bigfred/version"
printf '%s\n' "${COMMIT}" > "${TARGET_DIR}/usr/lib/bigfred/version/commit"
chmod 0644 "${TARGET_DIR}/usr/lib/bigfred/version/commit"
mkdir -p "${TARGET_DIR}/var/lib/bigfred"
echo "bigfred-os commit: ${COMMIT}"

SHORT=$(printf '%s' "${COMMIT}" | cut -c1-12)
[ -n "${SHORT}" ] || SHORT="unknown"

mkdir -p "${TARGET_DIR}/etc"
{
	echo "DISTRIB_ID=bigfred-os"
	echo "DISTRIB_RELEASE=rolling"
	echo "DISTRIB_CODENAME=hub"
	echo "DISTRIB_DESCRIPTION=\"BigFred OS (commit ${SHORT})\""
} > "${TARGET_DIR}/etc/lsb-release"
chmod 0644 "${TARGET_DIR}/etc/lsb-release"

# Overrides Buildroot's generic /etc/os-release.
{
	echo 'NAME="BigFred OS"'
	echo "ID=bigfred-os"
	echo 'VERSION_ID="rolling"'
	echo "VERSION=\"rolling (commit ${SHORT})\""
	echo "PRETTY_NAME=\"BigFred OS (commit ${SHORT})\""
	echo 'HOME_URL="https://github.com/dcc-bigfred/bigfred-os"'
} > "${TARGET_DIR}/etc/os-release"
chmod 0644 "${TARGET_DIR}/etc/os-release"
