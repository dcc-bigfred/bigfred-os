# Ochinchina supervisord for BigFred hub

Cross-builds [ochinchina/supervisord](https://github.com/ochinchina/supervisord)
plus a Python-compatible **supervisorctl** shim, and installs:

| Path | Role |
|------|------|
| `/usr/bin/supervisord` | ochinchina daemon |
| `/usr/bin/supervisorctl` | CLI shim → `supervisord -c CONF ctl …` |

`loco-server` and `bigfred-os-ui` only look for bare names on `PATH` — they do
not know about ochinchina. Same shim idea as
[deps-android-supervisord](https://github.com/dcc-bigfred/deps-android-supervisord)
(Android ships `lib*.so` and passes absolute paths).

## Configuration

In `make menuconfig` → *BigFred hub*:

- **ochinchina supervisord** — enable the package
- **ochinchina version** — default `0.7.3` (no leading `v`)

Enabled in `bigfred_hub_rpi5_defconfig` by default.

## Runtime

There is **no** SysV `S*-supervisord`. `loco-server` starts the daemon and
owns `/data/etc/supervisord/supervisord.conf` (HTTP ctl on `127.0.0.1:9001`).
