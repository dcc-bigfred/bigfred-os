# microdns (Buildroot package)

Pulls [`ghcr.io/dcc-bigfred/microdns-linux-arm64`](https://github.com/dcc-bigfred/microdns)
via `oras` and installs `/usr/sbin/microdns`.

Config is regenerated every boot from the rootfs overlay into
`/data/etc/microdns.json` (see `overlays/etc/microdns/microdns.json` and
`early-boot.sh`). The daemon is supervised by microinit (`/etc/init.d/microdns`).

## OCI tag

Default: `main` (also published as `sha-<7>` from microdns CI).

```bash
make image MICRODNS_OCI_TAG=main
# or pin: MICRODNS_OCI_TAG=sha-abcdef0
```

Menuconfig: `BR2_PACKAGE_MICRODNS_OCI_TAG`.

Host requirement: `oras` on `PATH` (`docker/install-buildroot-deps.sh`).
Optional auth: `GITHUB_TOKEN` / `GH_TOKEN` / `BIGFRED_NATIVE_TOKEN`.

Floating tags (`main`, `sha-*`) are re-pulled on each package download.
