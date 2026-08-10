# wireless-programmer (Buildroot package)

Fetches the prebuilt `wireless-programmer` linux/arm64 binary from
[dcc-bigfred/wireless-programmer](https://github.com/dcc-bigfred/wireless-programmer)
GitHub Actions artifacts (tip or Releases) and installs it as
`/usr/sbin/wireless-programmer`.

The daemon is driven over its Unix socket by `bigfred` / `bigfred-wizard`; see
`docs/api.md` in the source repo. Driving the radio (nl80211/rtnetlink) needs
`CAP_NET_ADMIN` + `CAP_NET_RAW`, currently satisfied by running the service as
root (see `os/overlays/etc/microinit.d/services/os/wireless-programmer.json`),
plus the kernel WiFi stack enabled (see `os/configs/linux-hub.fragment`).
