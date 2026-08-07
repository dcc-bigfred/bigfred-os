# microdns (Buildroot package)

Pulls [dcc-bigfred/microdns](https://github.com/dcc-bigfred/microdns) via
`go run github.com/dcc-bigfred/common/cmd/fetch@latest` (tip = Actions artifact;
`v*` / `latest-release` = GitHub Release) and installs `/usr/sbin/microdns`.

Config is regenerated every boot from the rootfs overlay into
`/data/etc/microdns.json` (see `overlays/etc/microdns/microdns.json` and
`early-boot.sh`). The daemon is supervised by microinit via the OS drop-in
`/etc/microinit.d/services/os/microdns.json` (refreshed into `/data` every boot).

## Ref

Default: `main` (also published as `sha-<7>` from microdns CI).

```bash
make image MICRODNS_REF=main
# or pin: MICRODNS_REF=sha-abcdef0
```

Menuconfig: `BR2_PACKAGE_MICRODNS_REF`.

Requires Go on the host (`install-go.sh` / Docker image). Tip refs need a
GitHub token in the environment.
