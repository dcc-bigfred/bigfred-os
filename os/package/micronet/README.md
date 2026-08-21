# micronet (Buildroot package)

Pulls the [dcc-bigfred/micronet](https://github.com/dcc-bigfred/micronet)
daemon via `go run github.com/dcc-bigfred/common/cmd/fetch@latest`
(tip = Actions artifact; `v*` / `latest-release` = GitHub Release) and
installs:

- `/usr/sbin/micronet`
- `/usr/sbin/configure-ethernet` and `/usr/sbin/configure-dhcp` as
  symlinks to `micronet` (one-release argv0 aliases)

Runtime on the image: `/usr/sbin/dnsmasq`, `/sbin/dhclient`, `/sbin/ip`
(`select` in `os/package/Config.in`). `microinit stop network` runs
`micronet teardown` (full cleanup of managed DHCP/client state).

Requires Go on the host (`install-go.sh` / Docker image). Tip refs need a
GitHub token in the environment.

Default ref: `main` (`MICRONET_REF` / `BR2_PACKAGE_MICRONET_REF`).
Fetch requires artifact `micronet-linux-arm64` (no legacy-name fallback).
Override: `make image MICRONET_REF=…`.

Default subnet seed: overlay `etc/micronet/micronet.json` (`192.168.0.1/24`),
copied to `$DATA_DIR/etc/micronet.json` on first boot only.
