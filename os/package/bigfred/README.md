# BigFred (`loco-server`)

Builds [dcc-bigfred/bigfred](https://github.com/dcc-bigfred/bigfred) from a
GitHub archive (branch or tag; Buildroot fetches `.tar.gz`, same tree as the
zip) and installs:

| Path | Role |
|------|------|
| `/opt/bigfred/bin/bigfred` | Cross-compiled `loco-server` binary (`dcc-bus` is a subcommand) |
| `/usr/bin/bigfred` | Wrapper: prefers `/data/opt/bigfred`, then `/opt/bigfred` |

## Configuration

In `make menuconfig` → *BigFred hub*:

- **BigFred (loco-server)** — enable the package
- **Git ref (branch or tag)** — default `master`

Examples: `master`, `v1.2.3`, `feat/my-branch`.

After changing the ref, clear the download cache for this package so Buildroot
does not reuse a stale archive:

```bash
rm -rf output/build/bigfred-* dl/bigfred
make bigfred-dirclean   # if the package was already built
make image
```

## Runtime override

Drop an updated binary on the RW data partition without reflashing:

```text
/data/opt/bigfred/bin/bigfred
```

`/usr/bin/bigfred` will pick it up before `/opt/bigfred/bin/bigfred`.

The image binary is built like `make server-build` (no embedded SPA). For a
`-tags prod` binary with `web/dist` embedded, build locally and install to
`/data/opt/bigfred/bin/bigfred`, or use `BIGFRED_OVERRIDE_SRCDIR` with a tree
that already has `web/dist` and set `BIGFRED_TAGS=prod` in a local override.

## Local source (no download)

```bash
make bigfred-override BIGFRED_OVERRIDE_SRCDIR=/path/to/bigfred
# or:
make BIGFRED_OVERRIDE_SRCDIR=/path/to/bigfred bigfred-rebuild
```
