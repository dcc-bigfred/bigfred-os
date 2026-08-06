# micronet (Buildroot package)

Pulls [`ghcr.io/dcc-bigfred/micronet-linux-arm64`](https://github.com/dcc-bigfred/micronet)
via ORAS and installs:

- `/usr/sbin/configure-ethernet` — club Ethernet bring-up (`network` / microinit)
- `/usr/sbin/configure-dhcp` — event WiFi DHCP when Omada (or other stack) detected

Default OCI tag: `main` (override: `make image MICRONET_OCI_TAG=…` or
`BR2_PACKAGE_MICRONET_OCI_TAG`).

Source: https://github.com/dcc-bigfred/micronet
