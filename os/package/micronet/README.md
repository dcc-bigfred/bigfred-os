# micronet (Buildroot package)

Pulls the [dcc-bigfred/micronet](https://github.com/dcc-bigfred/micronet)
daemon via `go run github.com/dcc-bigfred/common/cmd/fetch@latest`
(tip = Actions artifact; `v*` / `latest-release` = GitHub Release) and
installs:

- `/usr/sbin/micronet`
- `/usr/sbin/configure-ethernet` and `/usr/sbin/configure-dhcp` as
  symlinks to `micronet` (one-release argv0 aliases)

Requires Go on the host (`install-go.sh` / Docker image). Tip refs need a
GitHub token in the environment.

Default ref: `main` (`MICRONET_REF` / `BR2_PACKAGE_MICRONET_REF`).

Event subnet seed: overlay `etc/micronet/micronet.json` (`10.0.10.1/24`),
copied to `$DATA_DIR/etc/micronet.json` on first boot only.
