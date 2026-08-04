# microinit (Buildroot package)

Pulls [`ghcr.io/dcc-bigfred/microinit-linux-arm64`](https://github.com/dcc-bigfred/microinit)
via `oras` and installs:

- `/sbin/init` — PID 1
- `/usr/sbin/microinit` — same binary for CLI (`microinit list`, `start`, …)
- `/usr/sbin/shutdown` (+ `/sbin/shutdown` symlink) — SysV-style ordered
  poweroff/reboot/halt over the microinit control socket (when present in the
  OCI bundle)

Hub-specific early-boot and unmount stay in the rootfs overlay:
`overlays/etc/microinit/early-boot.sh` → `/etc/microinit/early-boot.sh`,
`overlays/etc/microinit/unmount.sh` → `/etc/microinit/unmount.sh`.
The portable `early-boot.sh` / `unmount.sh` layers from the OCI artifact are
**not** installed.

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
