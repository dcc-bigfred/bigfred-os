# bigfred (Buildroot package)

Pulls [dcc-bigfred/bigfred](https://github.com/dcc-bigfred/bigfred) hub linux/arm64
binaries via shared GitHub fetch and installs loco-server + remote-icmp under
`/opt/bigfred/bin/` with `/usr/bin` wrappers.

Tip refs (`master` / `main` / `sha-*`) use the CI artifact `binaries-arm64`.
Default ref: `master` (repo-root `VERSIONS` / `BIGFRED_REF`). Tip/`master`
needs a GitHub token; public Releases usually do not.

Requires Go on the host (`go run github.com/dcc-bigfred/common/cmd/fetch@latest`).
Every `make image` re-fetches the binaries (same as microinit/micronet/microdns).
