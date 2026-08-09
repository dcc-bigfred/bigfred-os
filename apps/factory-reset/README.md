# factory-reset

Destructive operator tool: wipe the local NVMe disk on a BigFred hub, write a
fresh GPT with one Linux FS partition, and format it as ext4
(`LABEL=bigfred-data`).

Does **not** copy `/data` and does **not** write the `.bigfred-nvme` migration
marker. After a wipe, the next boot falls back to microSD `/data`. Re-run
[`prepare-nvme`](../prepare-nvme/) to migrate again.

Linux only. Installed to `/usr/sbin/factory-reset` by `post-build.d/20-apps.sh`.

## Guards

Refuses to run unless:

1. `/etc/lsb-release` contains `DISTRIB_ID=bigfred-os`
2. `/var/lib/bigfred` exists (image marker from `post-build.d/60-identity.sh`)

## Usage

```bash
/usr/sbin/factory-reset --dry-run   # print plan, no disk changes
/usr/sbin/factory-reset             # interactive: type "yes" to confirm
/usr/sbin/factory-reset --yes       # skip confirmation (scripts / recovery)
```

## Build / test

```bash
make -C apps factory-reset          # cross → apps/.bin/factory-reset (aarch64 musl)
make -C apps/factory-reset test     # host unit tests
```

Shares NVMe helpers with `prepare-nvme` via a path dependency on that crate's
library (`prepare_nvme`).
