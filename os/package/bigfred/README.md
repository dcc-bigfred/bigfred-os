# bigfred (Buildroot package)

Pulls [dcc-bigfred/bigfred](https://github.com/dcc-bigfred/bigfred) hub linux/arm64
binaries via shared GitHub fetch and installs loco-server + remote-icmp under
`/opt/bigfred/bin/` with `/usr/bin` wrappers.

Default ref: `latest-release` (falls back to `master` if no release exists).
Tip/`master` needs a GitHub token; public Releases usually do not.

Requires `make ci-scripts` (`dcc-bigfred/.github` @ v2).
