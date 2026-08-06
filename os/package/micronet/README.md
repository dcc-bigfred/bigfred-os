# micronet (Buildroot package)

Pulls [dcc-bigfred/micronet](https://github.com/dcc-bigfred/micronet) binaries via
shared `fetch-github-binaries.sh` (tip = Actions artifact; `v*` / `latest-release`
= GitHub Release) and installs:

- `/usr/sbin/configure-ethernet`
- `/usr/sbin/configure-dhcp`

Requires `make ci-scripts` (clones `dcc-bigfred/.github` @ v2). Tip refs need a
GitHub token in the environment.

Default ref: `main` (`MICRONET_OCI_TAG` / `BR2_PACKAGE_MICRONET_OCI_TAG`).
