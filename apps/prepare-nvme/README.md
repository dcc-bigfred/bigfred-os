# prepare-nvme

Migrates hub writable state from microSD `/data` onto a local NVMe partition
and remounts `/data` there. Goal: keep the SD card read-only so it does not
wear out from continuous writes (logs, Redis, metrics, configs).

Written in **Rust**. GPT creation uses [`gptman`](https://crates.io/crates/gptman)
(pure Rust + Linux `BLKRRPART` ioctl). Format / mount / copy still use image
helpers (`mkfs.ext4`, `mount`, `cp`, `blkid`, optional `partprobe`).

Linux only. Installed to `/usr/sbin/prepare-nvme` by `post-build.sh`.

Refuses to run unless `/var/lib/bigfred` exists (OS marker created at image
build time alongside `/usr/lib/bigfred/version/commit`).

## Why

The hub boots from microSD (boot + rootfs + a small `/data` partition). Runtime
writes (logs, Redis, Grafana, Alloy, BigFred state) would grind that SD quickly.
With an NVMe stick present, early-boot runs this tool so `/data` lives on NVMe
and the SD root stays remounted read-only.

## Stages

| Stage | Command | Behaviour |
|-------|---------|-----------|
| 1 | `prepare` | Find `/dev/nvme*n*`. If no partition → write protective MBR + GPT with one Linux FS partition spanning the disk (`gptman`), format ext4. If the first partition exists and is empty (no files, or only `lost+found`) → format ext4. Then `cp -a` from `/data` onto the new filesystem and write a `.bigfred-nvme` marker at the partition root. |
| 2 | `mount` | If the first NVMe partition is ext4 → unmount current `/data` (SD) and mount the NVMe partition on `/data`. |

Default (`all`) runs prepare then mount. Missing NVMe is a no-op (exit 0).

The `.bigfred-nvme` marker tells `early-boot.sh` to mount the NVMe partition
directly on subsequent boots (no need to go through the SD first), so once
migrated the SD `/data` partition is never used again.

## Usage

```bash
/usr/sbin/prepare-nvme          # same as "all"
/usr/sbin/prepare-nvme prepare
/usr/sbin/prepare-nvme mount
```

Called from `/etc/microinit/early-boot.sh` after the SD `/data` mount and before
config seeding, so seeds and later services land on NVMe when available.

Verbose progress goes to stderr (`prepare-nvme: …`), including each external
command (`+ …`) and GPT decisions.

## Build / test

```bash
make -C apps prepare-nvme          # cross → apps/.bin/prepare-nvme (aarch64 musl)
make -C apps/prepare-nvme test     # host unit tests
```

Cross-link uses Buildroot’s `aarch64-buildroot-linux-musl-gcc` when
`os/output/host/` exists, otherwise `aarch64-linux-musl-gcc` on `PATH`.

## Requirements on the image

- `mkfs.ext4` (e2fsprogs) or BusyBox `mke2fs -t ext4`
- `blkid`, `mount` / `umount`, `cp`
- `partprobe` optional (fallback if `BLKRRPART` ioctl fails)
