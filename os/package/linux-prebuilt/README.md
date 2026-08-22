# linux-prebuilt (Buildroot package)

Installs the Raspberry Pi 5 kernel tarball published by
[dcc-bigfred/hub-kernel](https://github.com/dcc-bigfred/hub-kernel) Releases:

- `Image` + `bcm2712*-rpi-5-b.dtb` + `overlays/` → `output/images/` (boot)
- `/lib/modules/<krel>/` + `depmod` on the rootfs

Pin is **version + SHA-256** (`linux-prebuilt.hash`), same pattern as Grafana.
`BR2_DOWNLOAD_FORCE_CHECK_HASHES=y` skips the download when the cached tarball
matches the committed hash. Do not use `latest-release` at image-build time.

Kernel `CONFIG_*` (USB-serial, brcmfmac as a module, PREEMPT, 4K pages) lives
in hub-kernel, not in this tree.

Bump:

```bash
# After tagging a new hub-kernel release:
./scripts/sync-kernel.sh --bump v6.18.0-r2
```

Or edit `LINUX_PREBUILT_VERSION` in `linux-prebuilt.mk` and the `.hash` line.
