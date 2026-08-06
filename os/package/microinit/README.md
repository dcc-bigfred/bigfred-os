# microinit (Buildroot package)

Pulls [dcc-bigfred/microinit](https://github.com/dcc-bigfred/microinit) linux/arm64
binaries via shared GitHub fetch (not the distroless container image) and installs:

- `/usr/sbin/microinit` / `/sbin/init`
- `/usr/sbin/shutdown` when present in the artifact

Requires `make ci-scripts`. Tip refs need a GitHub token.

Default ref: `main`. Overlay scripts stay under `overlays/etc/microinit/`.
