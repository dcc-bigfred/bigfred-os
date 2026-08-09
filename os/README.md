# BigFred hub OS (Buildroot)

This directory is the **BR2_EXTERNAL** tree. BigFred (`loco-server`, `dcc-bus` as a
subcommand) is installed into the image via `package/bigfred`, which pulls hub
binaries from GitHub (default ref from repo-root `VERSIONS`, usually
`latest-release`). You can also install or override binaries under
`/data/opt/bigfred` after flash.

## Image contents

| Layer | Description |
|-------|-------------|
| **Bootloader / firmware** | `rpi-firmware`, `config.txt`, `cmdline.txt` (isolcpus, NVMe root) |
| **Kernel** | Raspberry Pi `linux` 6.6 (`bcm2712`) + USB-serial (cp210x, ftdi_sio, ch341, pl2303, cdc_acm) |
| **Rootfs** | BusyBox utilities, musl, RO `/`, RW `/data` (prefer NVMe via `prepare-nvme`); OS identity via `/etc/lsb-release` (`DISTRIB_ID=bigfred-os`), `/etc/os-release`, and `/usr/lib/bigfred/version/commit` |
| **Services** | **udevd** (eudev), Redis, SQLite, Grafana, VictoriaMetrics, bigfred-os-ui, Dropbear, watchdog, BigFred (`BR2_PACKAGE_BIGFRED`), optional Alloy |
| **Cooling** | Pi 5 active cooler via kernel `pwm-fan` (`dtparam=fan_temp*` in `board/bigfred_hub/config.txt`) |
| **Init** | **microinit** as `/sbin/init`; `/etc/init.d/*` scripts as backends; early-boot runs `fsck -y` then mounts `/data` |

## Host requirements

Same dependencies as the [Buildroot manual](https://buildroot.org/downloads/manual/manual.html#requirement)
(e.g. `gcc`, `make`, `ncurses`, `python3`, `rsync`, `wget`, `bc`).

## Build

From the repository root (recommended):

```bash
make image                  # on host (requires Buildroot dependencies)
make image-using-docker     # Ubuntu 24.04 in Docker (uid/gid 1000:1000)
# Defaults live in repo-root VERSIONS. Optional pins:
# make image BIGFRED_REF=master MICROINIT_REF=main MICRONET_REF=main MICRODNS_REF=main
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
Rust for `prepare-nvme`: `sudo docker/install-rust.sh` (rustup +
`aarch64-unknown-linux-musl`), or use `make image-using-docker` (image includes it).

After changing the Docker image: `make docker-image`, then `make image-using-docker`.

### Host errors with GCC 15 (Manjaro/Arch)

GCC 15 defaults to `-std=gnu23`; older host packages may fail (`host-m4`,
`host-e2fsprogs`, …). We use Buildroot **2025.02** and C-only workarounds in
`os/external.mk` (not global `HOST_CFLAGS` — that breaks `host-gcc`). After changes, clean:

```bash
rm -rf os/output/build/host-m4-* os/output/build/host-e2fsprogs-*
make -C os image
```

### Docker `apparmor/profiles` on Manjaro/Arch

If `docker run` fails with:

```text
Could not check if docker-default AppArmor profile was loaded:
open /sys/kernel/security/apparmor/profiles: no such file or directory
```

the kernel has AppArmor but **securityfs** is not mounted (empty
`/sys/kernel/security/`). `make image-using-docker` auto-adds
`--security-opt apparmor=unconfined` when that path is missing.

To fix the host permanently:

```bash
sudo mount -t securityfs securityfs /sys/kernel/security
sudo systemctl restart apparmor docker
docker run --rm hello-world
```

For a persistent mount, ensure `securityfs` is mounted at boot (AppArmor
service or `/etc/fstab` entry for `/sys/kernel/security`).

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
3. **Time & timezone** — no RTC/NTP. Wall clock persists in `/data/etc/fake-hwclock`
   (restored in `early-boot.sh`, refreshed by cron every 10 min). Timezone via
   `/data/etc/timezone` + bind of `/data/etc/localtime` over RO `/etc/localtime`
   (default `Europe/Warsaw`). UI: **Time** tab; CLI: `bigfred-set-time`,
   `bigfred-set-timezone`.
4. **LocoNet USB** — kernel drivers in `configs/linux-hub.fragment`
   (`cp210x`, `ftdi_sio`, `ch341`, `pl2303`, `cdc_acm`); **udevd** microinit
   service applies `overlays/etc/udev/rules.d/99-loconet-usb.rules`
   (symlinks `/dev/loconet-63120`, `loconet-lb-usb`, `loconet-ch340`).
   Centrals stub: `99-loconet-centrals.rules`. See §USB / udev below.
5. **PREEMPT_RT** — `configs/linux-hub.fragment`; if the kernel build fails, use an
   RT tag/branch from `raspberrypi/linux` or temporarily remove `CONFIG_PREEMPT_RT=y`.
6. **Grafana Alloy** — enabled in defconfig (`BR2_PACKAGE_ALLOY`); built from
   source as a static musl-compatible binary (official release zips are
   glibc-only). Config: `overlays/etc/alloy/config.alloy`.
7. **Pi 5 Rev 1.1 (BCM2712 D0)** — `board/bigfred_hub/config.txt` sets
   `device_tree=bcm2712d0-rpi-5-b.dtb`. Without it, D0 boards panic in
   `bcm2712_pull_config_set` / `brcmuart_init`. For older Rev 1.0 (C0) use
   `bcm2712-rpi-5-b.dtb` instead.

## USB / udev (LocoNet)

`BR2_INIT_NONE` means Buildroot's `S10udev` never runs. The hub starts **eudev**
via microinit service `udevd` (`overlays/etc/init.d/udevd`) so `SYMLINK` /
`GROUP` / `MODE` rules apply on coldplug and hotplug. `bigfred` depends on
`udevd`.

| Symlink | Match | Hardware |
|---------|-------|----------|
| `/dev/loconet-63120` | `10c4:ea60` | Uhlenbrock 63120 (CP210x → `ttyUSB*`) |
| `/dev/loconet-lb-usb` | `0403:c7d0` | RR-CirKits LocoBuffer-USB |
| `/dev/loconet-ch340` | `1a86:7523` | DIY CH340 |
| (raw) `ttyACM*` | dialout | Digitrax PR3/PR4 (cdc_acm); RB1110 ACM is LI100F — use Z21 UDP |

Kernel options: `CONFIG_USB_SERIAL_CP210X`, `FTDI_SIO`, `CH341`, `PL2303`,
`CONFIG_USB_ACM`. On device: `microinit list \| grep udevd`,
`ls -l /dev/loconet-*`.

## BigFred (loco-server)

Buildroot package `package/bigfred` pulls hub linux/arm64 binaries from GitHub
via `go run github.com/dcc-bigfred/common/cmd/fetch@latest` and installs:

- `/opt/bigfred/bin/bigfred` — binary (`dcc-bus` is a subcommand of the same binary)
- `/opt/bigfred/bin/bigfred-remote-icmp`
- `/usr/bin/bigfred` — wrapper: prefers `/data/opt/bigfred`, then `/opt/bigfred`

Ref: defaults in repo-root `VERSIONS`; override with
`make image BIGFRED_REF=master` (or `latest-release`, `v1.2.3`).
Menuconfig: `BR2_PACKAGE_BIGFRED_REF`. Requires Go on the host (see
`docker/install-buildroot-deps.sh`). Details: `package/bigfred/README.md`.

## microinit (PID 1)

Buildroot package `package/microinit` pulls linux/arm64 binaries from GitHub
via the same fetch tool and installs `/sbin/init`,
`/usr/sbin/microinit`, and `/usr/sbin/shutdown` (SysV-style ordered
poweroff/reboot/halt). Init system is `BR2_INIT_NONE` (BusyBox stays for
utilities; it does not own `/sbin/init`).

- Early-boot: `overlays/etc/microinit/early-boot.sh` (`fsck -y` before mounts,
  mounts `/data`, optional `prepare-nvme` migrate to NVMe so the microSD stays
  read-only, seeds configs)
- Unmount: `overlays/etc/microinit/unmount.sh` (unbind shadow, remount/umount
  `/data`, remount root RO, `umount -a -r`; called at end of shutdown)

- Service list seed: `overlays/etc/microinit/microinit.json` → `/data/etc/microinit.json`
  (includes **udevd** early; `bigfred` `dependsOn` includes `udevd`)
- OS drop-in `orderPriority` bands: see
  `overlays/etc/microinit.d/services/os/README.md`
- Scripts under `overlays/etc/init.d/` remain as `cmd` backends

Ref: defaults in repo-root `VERSIONS`; override with
`make image MICROINIT_REF=main` (or `sha-<7>`, `v*`, `latest-release`).
Menuconfig: `BR2_PACKAGE_MICROINIT_REF`. Details: `package/microinit/README.md`.

CLI on device: `microinit list`, `microinit start redis`, `microinit logs --follow`.

Operator disk tools (installed under `/usr/sbin/`):

| Tool | Role |
|------|------|
| `prepare-nvme` | Migrate `/data` from microSD onto NVMe (safe/empty format + copy) |
| `factory-reset` | Destructive NVMe wipe + GPT/ext4 reformat (requires `DISTRIB_ID=bigfred-os`) |
| `bigfred-set-time` / `bigfred-set-timezone` | Persist time/timezone under `/data` |

Init scripts: `bigfred` with `taskset` on cores 2,3. `remote-icmp` starts the
ICMP helper.

Databases: SQLite `/data/var/db/bigfred/bigfred.sqlite3`, Redis `/data/var/db/redis/`
(config `/data/etc/redis.conf`, default RDB `save 60 100`).

Service accounts (non-root): `redis`, `alloy`, `bigfred` (also in `dialout`),
`metrics` (VictoriaMetrics + Grafana). `bigfred-os-ui` stays root.
`bigfred` home is `/home/bigfred` (tmpfs, no login shell). microinit IPC:
`socketAllowUsers: ["bigfred"]` (socket `0660`).

microinit drop-in layout under `/data/etc/microinit.d/` (set each boot by
early-boot): parents `microinit.d/` and `services/` are `root:bigfred` mode
`0750` (loco-server may traverse, not create/delete sibling groups);
`services/infra/` and `services/dcc-bus/` are `bigfred:bigfred` `0750`
(loco-server writes drop-ins); `services/os/` stays `root:root` `0755`
(image-managed).

Monitoring: Grafana (`http://:3000`, admin/bigfred) with VictoriaMetrics datasource
(`:8428`). Data: `/data/var/lib/grafana`, `/data/var/lib/victoriametrics`,
Alloy `/data/var/lib/alloy`.
VM disk flush: `-inmemoryDataFlushInterval` in `/etc/default/victoriametrics`.

Admin panel: `bigfred-os-ui` (`http://:8090`, config in `/data/etc/bigfred-os-ui.conf`).

## Layout

```text
os/
├── configs/           # defconfig, kernel and BusyBox fragments
├── board/bigfred_hub/ # cmdline, config.txt, genimage, post-build.sh + post-build.d/
├── overlays/          # fstab, init.d, redis, crontab, udev
├── kernel/            # (fragments in configs/linux-hub.fragment)
├── package/           # bigfred, alloy, grafana, victoriametrics (hub apps: ../apps/)
├── scripts/           # flash-nvme.sh
../apps/                 # apps → apps/.bin/ → /usr/sbin/ (Go + Rust prepare-nvme)
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
