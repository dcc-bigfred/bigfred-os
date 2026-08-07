# micronet (Buildroot package)

Pulls [dcc-bigfred/micronet](https://github.com/dcc-bigfred/micronet) binaries via
`go run github.com/dcc-bigfred/common/cmd/fetch@latest` (tip = Actions artifact;
`v*` / `latest-release` = GitHub Release) and installs:

- `/usr/sbin/configure-ethernet`
- `/usr/sbin/configure-dhcp`

Requires Go on the host (`install-go.sh` / Docker image). Tip refs need a
GitHub token in the environment.

Default ref: `main` (`MICRONET_REF` / `BR2_PACKAGE_MICRONET_REF`).
