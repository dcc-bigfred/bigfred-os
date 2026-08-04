#!/usr/bin/env bash
# Install Rust (rustup) + aarch64-unknown-linux-musl target for prepare-nvme.
# Used by docker/Dockerfile and optionally on bare hosts.
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	exec sudo --preserve-env=DEBIAN_FRONTEND,RUSTUP_HOME,CARGO_HOME,RUST_VERSION "$0" "$@"
fi

export DEBIAN_FRONTEND="${DEBIAN_FRONTEND:-noninteractive}"
export RUSTUP_HOME="${RUSTUP_HOME:-/usr/local/rustup}"
export CARGO_HOME="${CARGO_HOME:-/usr/local/cargo}"
RUST_VERSION="${RUST_VERSION:-stable}"

mkdir -p "${RUSTUP_HOME}" "${CARGO_HOME}"

# curl | sh is the supported rustup installer; pin via RUSTUP_INIT_SKIP_PATH_CHECK.
echo "Installing rustup (${RUST_VERSION}) → RUSTUP_HOME=${RUSTUP_HOME} CARGO_HOME=${CARGO_HOME} ..."
curl --proto '=https' --tlsv1.2 -fsSL https://sh.rustup.rs \
	| sh -s -- -y --no-modify-path \
		--default-toolchain "${RUST_VERSION}" \
		--profile minimal \
		-c rustfmt \
		-t aarch64-unknown-linux-musl

# Stable links on PATH (same pattern as install-go.sh).
ln -sfn "${CARGO_HOME}/bin/cargo" /usr/local/bin/cargo
ln -sfn "${CARGO_HOME}/bin/rustc" /usr/local/bin/rustc
ln -sfn "${CARGO_HOME}/bin/rustup" /usr/local/bin/rustup

# Image runs as uid 1000 — allow crate downloads / incremental builds.
chown -R 1000:1000 "${RUSTUP_HOME}" "${CARGO_HOME}" 2>/dev/null || true

cargo --version
rustc --version
rustup target list --installed
