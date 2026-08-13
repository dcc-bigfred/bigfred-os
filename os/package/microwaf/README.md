# microwaf (Buildroot package)

Fetches the prebuilt `microwaf` linux/arm64 binary from
[dcc-bigfred/microwaf](https://github.com/dcc-bigfred/microwaf)
GitHub Actions artifacts (tip or Releases) and installs it as
`/usr/sbin/microwaf`.

Host-side WAF (HTTP / WebSocket / Z21 / WiThrottle + generic UDP/TCP).
Sniffs via AF_PACKET; optional XDP enforce. Config is seeded on first
start under `/data/etc/microwaf/`. Socket: `/data/run/microwaf/microwaf.sock`.

The microinit drop-in is **disabled by default** (`enabled: false`) so the
daemon is on the image but does not start at boot.

```
# until next reboot (OS drop-ins are refreshed from the image every boot)
microinit start microwaf
```

Persistent enable: set `"enabled": true` in
`overlays/etc/microinit.d/services/os/microwaf.json` and rebuild the image.

Needs root (`CAP_NET_RAW` / `CAP_NET_ADMIN` / BPF). Kernel options:
`os/configs/linux-hub.fragment`.
