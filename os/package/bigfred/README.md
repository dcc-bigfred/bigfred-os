# BigFred (`loco-server`)

Fetches the hub OCI bundle from GHCR
([`ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64`](https://github.com/dcc-bigfred/bigfred))
via `oras` and installs:

| Path | Role |
|------|------|
| `/opt/bigfred/bin/bigfred` | `loco-server` (`dcc-bus` is a subcommand) |
| `/opt/bigfred/bin/bigfred-remote-icmp` | Wi-Fi RTT probe helper |
| `/usr/bin/bigfred` | Wrapper: prefers `/data/opt/bigfred`, then `/opt/bigfred` |

Bundle tags (from the `bigfred` CI):

| Tag | Meaning |
|-----|---------|
| `latest-release` | Latest `v*` release (default) |
| `master` | Tip of `master` |
| `v1.2.3` | Pinned release |
| `sha-<7>` | Immutable commit from a `master` push |

## Configuration

Makefile (preferred):

```bash
make image BIGFRED_OCI_TAG=latest-release
make image BIGFRED_OCI_TAG=master
make image-using-docker BIGFRED_OCI_TAG=v1.2.3
```

Or in `make menuconfig` → *BigFred hub* → **OCI tag**.

Host requirement: `oras` on `PATH` (installed by `docker/install-buildroot-deps.sh`).
For private GHCR packages, export `GITHUB_TOKEN` (or `GH_TOKEN`).

Floating tags (`master`, `latest-release`, `sha-*`) are re-pulled on every package
download. After changing a pinned `v*` tag, clear the cache if needed:

```bash
rm -rf os/output/build/bigfred-* os/dl/bigfred
make -C os bigfred-dirclean   # if already built
make image BIGFRED_OCI_TAG=v1.2.3
```

## Runtime override

Drop an updated binary on the RW data partition without reflashing:

```text
/data/opt/bigfred/bin/bigfred
```

`/usr/bin/bigfred` (and `S60-bigfred` / Update tab) prefer `/data/opt` over `/opt`.

## Init

`overlays/etc/init.d/S60-bigfred` starts loco-server with CPU affinity on cores 2,3.
`S41-remote-icmp` starts the ICMP helper separately.
