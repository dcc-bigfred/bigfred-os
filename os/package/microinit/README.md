# microinit (Buildroot package)

Pulls [`ghcr.io/dcc-bigfred/microinit-linux-arm64`](https://github.com/dcc-bigfred/microinit)
via `oras` and installs:

- `/sbin/init` — PID 1
- `/usr/sbin/microinit` — same binary for CLI (`microinit list`, `start`, …)

Hub-specific early-boot stays in the rootfs overlay:
`overlays/etc/microinit/early-boot.sh` → `/etc/microinit/early-boot.sh`.
The portable `early-boot.sh` layer from the OCI artifact is **not** installed.

## OCI tag

Default: `main` (also published as `sha-<7>` from microinit CI).

```bash
make image MICROINIT_OCI_TAG=main
# or pin: MICROINIT_OCI_TAG=sha-abcdef0
```

Menuconfig: `BR2_PACKAGE_MICROINIT_OCI_TAG`.

Host requirement: `oras` on `PATH` (`docker/install-buildroot-deps.sh`).
Optional auth: `GITHUB_TOKEN` / `GH_TOKEN` / `BIGFRED_NATIVE_TOKEN`.

Floating tags (`main`, `sha-*`) are re-pulled on each package download.
