# factory-reset

Operator tool: wipe hub writable state under `/data` and reboot.

`/data` itself cannot be unmounted while services hold it. Redis is stopped
best-effort first (via `microinit stop redis`, falling back to `killall`) so its
SIGTERM save handler cannot recreate `dump.rdb` under `/data`. Nested mounts
under `/data/…` (notably tmpfs `/data/logs`) are unmounted next, then every
entry under `/data` is deleted. Finally `shutdown -r now` so early-boot reseeds
the layout on the next boot.

Linux only. Installed to `/usr/sbin/factory-reset` by `post-build.d/20-apps.sh`.

## Guards

Refuses to run unless:

1. `/etc/lsb-release` contains `DISTRIB_ID=bigfred-os`
2. `/var/lib/bigfred` exists (image marker from `post-build.d/60-identity.sh`)

## Usage

```bash
/usr/sbin/factory-reset --dry-run   # print plan, no changes
/usr/sbin/factory-reset             # interactive: type "yes" to confirm
/usr/sbin/factory-reset --yes       # skip confirmation (scripts / recovery)
```

## Build / test

```bash
make -C apps factory-reset          # cross → apps/.bin/factory-reset (aarch64 musl)
make -C apps/factory-reset test     # host unit tests
```
