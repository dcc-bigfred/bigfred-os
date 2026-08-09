# wireless-programmer (Buildroot package)

Fetches the prebuilt `wireless-programmer` linux/arm64 binary from
[ dcc-bigfred/wireless-programmer ](https://github.com/dcc-bigfred/wireless-programmer)
GitHub Actions artifacts (tip or Releases) and installs it as
`/usr/sbin/wireless-programmer`.

The daemon is driven over its Unix socket by `bigfred` / `bigfred-wizard`; see
`docs/api.md` in the source repo. It needs `CAP_NET_ADMIN` + `CAP_NET_RAW`
to drive the radio (nl80211/rtnetlink) and the kernel WiFi stack enabled
(see `os/configs/linux-hub.fragment`).
