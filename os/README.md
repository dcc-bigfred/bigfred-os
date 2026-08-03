# BigFred hub OS (Buildroot)

This directory is the **BR2_EXTERNAL** tree. BigFred (`loco-server`, `dcc-bus` as a
subcommand) is installed into the image via `package/bigfred`, which pulls the
hub OCI bundle from GHCR (`BIGFRED_OCI_TAG`, default `latest-release`). You can
also install or override binaries under `/data/opt/bigfred` after flash.

## Image contents

| Layer | Description |
|-------|-------------|
| **Bootloader / firmware** | `rpi-firmware`, `config.txt`, `cmdline.txt` (isolcpus, NVMe root) |
| **Kernel** | Raspberry Pi `linux` 6.6 (`bcm2712`) + RT and USB-ACM fragments |
| **Rootfs** | BusyBox utilities, musl, RO `/`, RW `/data` |
| **Services** | Redis, SQLite, Grafana, VictoriaMetrics, bigfred-os-ui, Dropbear, watchdog, fanctl, BigFred (`BR2_PACKAGE_BIGFRED`), optional Alloy |
| **Init** | **microinit** as `/sbin/init` (OCI); SysV `S??-*` scripts as backends; early-boot mounts `/data` |

## Host requirements

Same dependencies as the [Buildroot manual](https://buildroot.org/downloads/manual/manual.html#requirement)
(e.g. `gcc`, `make`, `ncurses`, `python3`, `rsync`, `wget`, `bc`).

## Build

From the repository root (recommended):

```bash
make image                  # on host (requires Buildroot dependencies)
make image-using-docker     # Ubuntu 24.04 in Docker (uid/gid 1000:1000)
# Optional OCI pins:
# make image BIGFRED_OCI_TAG=master MICROINIT_OCI_TAG=main
```

Docker mounts the repo at the **same absolute path** as on the host. Host tools with
`$ORIGIN/../lib` are fine; `rm -rf os/output` is only needed when RUNPATH points at a
**different** stale absolute path (e.g. after building from `/work` instead of the full
host path).

Or build only the OS layer:

```bash
cd os
make image
```

Host dependencies (Ubuntu/Debian): `sudo docker/install-buildroot-deps.sh`
(includes `flex`/`libfl2` — cross-`ar` from binutils links `libfl.so.2`).

After changing the Docker image: `make docker-image`, then `make image-using-docker`.

### Host errors with GCC 15 (Manjaro/Arch)

GCC 15 defaults to `-std=gnu23`; older host packages may fail (`host-m4`,
`host-e2fsprogs`, …). We use Buildroot **2025.02** and C-only workarounds in
`os/external.mk` (not global `HOST_CFLAGS` — that breaks `host-gcc`). After changes, clean:

```bash
rm -rf os/output/build/host-m4-* os/output/build/host-e2fsprogs-*
make -C os image
```

### GitHub Actions (manual)

The **Build hub OS image** workflow (`/.github/workflows/build-hub-os.yml`) caches
downloads (`os/buildroot/dl`), the Buildroot tree, and the **host toolchain**
(`os/.cache/host-toolchain` — `host-gcc`, musl, `output/host`). The *clean* option
clears the toolchain cache.

1. Repository → **Actions** → **Build hub OS image** → **Run workflow**
2. Options: *clean* (full rebuild), *skip_tests*
3. When finished (~1–3 h): artifact `bigfred-hub-nvme-<run>` with `hub-nvme.img` and SHA-256 sum

The first local run downloads Buildroot **2025.02** and builds the image (slow,
depending on CPU and cache).

Output:

```text
output/images/hub-nvme.img
output/images/sdcard.img   # symlink
```

## Flash to SD card

From the repository root (after `make image`):

```bash
sudo ./scripts/flash-sdcard.sh
# optional: sudo ./scripts/flash-sdcard.sh os/output/images/hub-nvme.img
```

The script only lists `mmcblk*` whole disks, lets you pick a device, and requires
typing `YES` before writing.

## Flash to NVMe

From the `os/` directory:

```bash
sudo ./scripts/flash-nvme.sh /dev/nvme0n1 output/images/hub-nvme.img
```

## Configuration before deployment

1. **Network** — `board/bigfred_hub/network.conf` (copied to `/etc/bigfred/network.conf`).
2. **Root password** — default `root` in defconfig; change via `make menuconfig`
   → *System configuration* → *Root password*, or on device: `passwd root`
   (password in `/data/etc/shadow`, **Account** panel in `bigfred-os-ui`).
3. **Uhlenbrock 63120** — update `overlays/etc/udev/rules.d/99-uhlenbrock-63120.rules`
   after `udevadm` (§3.5).
4. **PREEMPT_RT** — `configs/linux-hub.fragment`; if the kernel build fails, use an
   RT tag/branch from `raspberrypi/linux` or temporarily remove `CONFIG_PREEMPT_RT=y`.
5. **Grafana Alloy** — enabled in defconfig (`BR2_PACKAGE_ALLOY`); binary is fetched
   from GitHub releases at build time. Config: `overlays/etc/alloy/config.alloy`.
6. **Pi 5 Rev 1.1 (BCM2712 D0)** — `board/bigfred_hub/config.txt` sets
   `device_tree=bcm2712d0-rpi-5-b.dtb`. Without it, D0 boards panic in
   `bcm2712_pull_config_set` / `brcmuart_init`. For older Rev 1.0 (C0) use
   `bcm2712-rpi-5-b.dtb` instead.

## BigFred (loco-server)

Buildroot package `package/bigfred` pulls
`ghcr.io/dcc-bigfred/bigfred-hub-linux-arm64` (ORAS) and installs:

- `/opt/bigfred/bin/bigfred` — binary (`dcc-bus` is a subcommand of the same binary)
- `/opt/bigfred/bin/bigfred-remote-icmp`
- `/usr/bin/bigfred` — wrapper: prefers `/data/opt/bigfred`, then `/opt/bigfred`

OCI tag: `make image BIGFRED_OCI_TAG=master` (or `latest-release`, `v1.2.3`).
Menuconfig: `BR2_PACKAGE_BIGFRED_OCI_TAG`. Requires host `oras` (see
`docker/install-buildroot-deps.sh`). Details: `package/bigfred/README.md`.

## microinit (PID 1)

Buildroot package `package/microinit` pulls
`ghcr.io/dcc-bigfred/microinit-linux-arm64` (ORAS) and installs `/sbin/init`
plus `/usr/sbin/microinit`. Init system is `BR2_INIT_NONE` (BusyBox stays for
utilities; it does not own `/sbin/init`).

- Early-boot: `overlays/etc/microinit/early-boot.sh` (mounts `/data`, seeds configs)
- Service list seed: `overlays/etc/microinit/microinit.json` → `/data/etc/microinit.json`
- SysV scripts under `overlays/etc/init.d/S??-*` remain as `cmd` backends

OCI tag: `make image MICROINIT_OCI_TAG=main` (or `sha-<7>`).
Menuconfig: `BR2_PACKAGE_MICROINIT_OCI_TAG`. Details: `package/microinit/README.md`.

CLI on device: `microinit list`, `microinit start redis`, `microinit logs --follow`.

Init scripts: `S60-bigfred` with `taskset` on cores 2,3. `S41-remote-icmp` starts the
ICMP helper.

Databases: `/data/sqlite/`, Redis: `/data/redis/` (config `/data/etc/redis.conf`, default RDB `save 60 100`).

Monitoring: Grafana (`http://:3000`, admin/bigfred) with VictoriaMetrics datasource
(`:8428`). Data: `/data/opt/grafana`, `/data/opt/victoriametrics`.
VM disk flush: `-inmemoryDataFlushInterval=5m` in `/etc/default/victoriametrics`.

Admin panel: `bigfred-os-ui` (`http://:8090`, config in `/data/etc/bigfred-os-ui.conf`).

## Layout

```text
os/
├── configs/           # defconfig, kernel and BusyBox fragments
├── board/bigfred_hub/ # cmdline, config.txt, genimage, post-*.sh
├── overlays/          # fstab, init.d, redis, crontab, udev
├── kernel/            # (fragments in configs/linux-hub.fragment)
├── package/           # bigfred, alloy, grafana, victoriametrics (hub apps: ../apps/)
├── scripts/           # flash-nvme.sh
../apps/                 # Go apps → apps/.bin/ → /usr/sbin/ on image
../scripts/              # flash-sdcard.sh (repo root)
├── Makefile
└── external.desc
```

## Customization

```bash
make menuconfig    # Buildroot packages
make image
```

Project defconfig: `configs/bigfred_hub_rpi5_defconfig`.
