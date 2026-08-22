# Kernel for the hub image

The Pi 5 kernel is **not** built in this tree. `package/linux-prebuilt`
downloads a hash-pinned tarball from
[dcc-bigfred/hub-kernel](https://github.com/dcc-bigfred/hub-kernel) Releases
(`Image`, DTBs, overlays, modules). Pin: `LINUX_PREBUILT_VERSION` +
`linux-prebuilt.hash`.

Kconfig fragments (`linux-hub.fragment`, 4K pages, brcmfmac as a module)
live in hub-kernel. A `CONFIG_*` change is a PR there, then a Release tag,
then:

```bash
./scripts/sync-kernel.sh --bump v6.18.0-rN
```
