# wireless-programmer (Buildroot package)

Fetches the prebuilt `wireless-programmer` linux/arm64 binary from
[dcc-bigfred/wireless-programmer](https://github.com/dcc-bigfred/wireless-programmer)
GitHub Actions artifacts (tip or Releases) and installs it as
`/usr/sbin/wireless-programmer`.

The daemon is driven over its Unix socket by `bigfred` / `bigfred-wizard`; see
`docs/api.md` in the source repo. Driving the radio (nl80211/rtnetlink) needs
`CAP_NET_ADMIN` + `CAP_NET_RAW`, currently satisfied by running the service as
root (see `os/overlays/etc/microinit.d/services/os/wireless-programmer.json`),
plus the kernel WiFi stack enabled (hub-kernel `linux-hub.fragment`) and
Pi 5 CYW43455 firmware from `BR2_PACKAGE_BRCMFMAC_SDIO_FIRMWARE_RPI_{WIFI,BT}`
(`brcmfmac43455-sdio.raspberrypi,5-model-b.{bin,txt,clm_blob}` and
`BCM4345C0.raspberrypi,5-model-b.hcd`). `brcmfmac` is a module loaded by
`udevd` after rootfs is mounted — a built-in probe runs before `/lib/firmware`
exists and never creates `wlan0`.
