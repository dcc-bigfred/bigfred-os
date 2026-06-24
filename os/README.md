# BigFred hub OS (Buildroot)

Obraz referencyjny dla **Raspberry Pi 5** opisany w dokumentacji
`modelarstwo/rb/docs/loconet-adapter` (rozdział **§8 Hub OS image**).

Ten katalog to drzewo **BR2_EXTERNAL** — nie zawiera binariów **BigFred**
(`loco-server`, `dcc-bus`, `web/dist`). Zainstalujesz je osobno (np. na
partycji `/` w trybie RW lub przez własny pakiet Buildroot).

## Co zawiera obraz

| Warstwa | Opis |
|---------|------|
| **Bootloader / firmware** | `rpi-firmware`, `config.txt`, `cmdline.txt` (isolcpus, NVMe root) |
| **Jądro** | Raspberry Pi `linux` 6.6 (`bcm2712`) + fragmenty RT i USB-ACM |
| **Rootfs** | BusyBox init, musl, RO `/`, RW `/data` |
| **Usługi** | Redis, SQLite, Grafana, VictoriaMetrics, bigfred-os-ui, Dropbear, watchdog, fanctl, opcjonalnie Alloy |
| **Init** | `S05`…`S95` (VictoriaMetrics `S35`, Grafana `S42`, bigfred-os-ui `S48`; bez `S60-bigfred`) |

## Wymagania hosta

Zależności jak w [manualu Buildroot](https://buildroot.org/downloads/manual/manual.html#requirement)
(m.in. `gcc`, `make`, `ncurses`, `python3`, `rsync`, `wget`, `bc`).

## Budowa

Z katalogu głównego repozytorium (zalecane):

```bash
make image                  # host (wymaga zależności Buildroot)
make image-using-docker     # Ubuntu 24.04 w Dockerze (uid/gid 1000:1000)
```

Docker montuje repo pod **tą samą ścieżką bezwzględną** co na hoście. Host tools z
`$ORIGIN/../lib` są OK; `rm -rf os/output` potrzebne tylko gdy RUNPATH wskazuje
**inną** starą ścieżkę absolutną (np. po buildzie z `/work` zamiast z pełnej ścieżki).

Albo tylko warstwa OS:

```bash
cd os
make image
```

Zależności hosta (Ubuntu/Debian): `sudo docker/install-buildroot-deps.sh`
(m.in. `flex`/`libfl2` — cross-`ar` z binutils linkuje `libfl.so.2`).

Po zmianie obrazu Docker: `make docker-image`, potem `make image-using-docker`.

### Błędy hosta przy GCC 15 (Manjaro/Arch)

GCC 15 domyślnie używa `-std=gnu23`; starsze host-pakiety mogą paść (`host-m4`,
`host-e2fsprogs`, …). Używamy Buildroot **2025.02** oraz obejść C-only w
`os/external.mk` (nie globalnego `HOST_CFLAGS` — psuje `host-gcc`). Po zmianie wyczyść:

```bash
rm -rf os/output/build/host-m4-* os/output/build/host-e2fsprogs-*
make -C os image
```

### GitHub Actions (ręcznie)

Workflow **Build hub OS image** (`/.github/workflows/build-hub-os.yml`) cache’uje m.in.
pobrania (`os/buildroot/dl`), drzewo Buildroot oraz **host toolchain**
(`os/.cache/host-toolchain` — `host-gcc`, musl, `output/host`). Przy opcji *clean*
cache toolchaina jest czyszczony.


1. Repozytorium → **Actions** → **Build hub OS image** → **Run workflow**
2. Opcje: *clean* (pełny rebuild), *skip_tests*
3. Po zakończeniu (~1–3 h): artefakt `bigfred-hub-nvme-<run>` z `hub-nvme.img` i sumą SHA-256

Lokalny flash: `sudo ./scripts/flash-nvme.sh /dev/nvme0n1 output/images/hub-nvme.img`

Pierwsze uruchomienie pobierze Buildroot `2024.11` i zbuduje obraz (długo,
zależnie od CPU i cache).

Wynik:

```text
output/images/hub-nvme.img
output/images/sdcard.img   # symlink
```

## Flash na NVMe

```bash
sudo ./scripts/flash-nvme.sh /dev/nvme0n1 output/images/hub-nvme.img
```

## Konfiguracja przed wdrożeniem

1. **Sieć** — `board/bigfred_hub/network.conf` (kopiowany do `/etc/bigfred/network.conf`).
2. **Hasło root** — domyślnie `root` w defconfig; zmień przez `make menuconfig`
   → *System configuration* → *Root password*, lub na urządzeniu: `passwd root`
   (hasło w `/data/etc/shadow`, panel **Account** w `bigfred-os-ui`).
3. **Uhlenbrock 63120** — uzupełnij `overlays/etc/udev/rules.d/99-uhlenbrock-63120.rules`
   po `udevadm` (§3.5).
4. **PREEMPT_RT** — fragment `configs/linux-hub.fragment`; jeśli kompilacja jądra
   się wyłoży, użyj tagu/branży `raspberrypi/linux` z RT lub tymczasowo usuń
   `CONFIG_PREEMPT_RT=y`.
5. **Grafana Alloy** — `make menuconfig` → włącz `BR2_PACKAGE_ALLOY` i umieść
   binarkę `package/alloy/alloy-linux-arm64`.

## Instalacja BigFred (poza tym repo)

Po flashu, z innego builda Go (`GOOS=linux GOARCH=arm64`):

- `/usr/bin/loco-server`, `/usr/bin/dcc-bus`
- `/usr/share/bigfred/web`
- skrypt init `S60-bigfred` (wzorzec w dokumentacji §8.3) z `taskset -c 2,3`

Bazy: `/data/sqlite/`, Redis: `/data/redis/` (config `/data/etc/redis.conf`, domyślnie RDB `save 60 100`).

Monitoring: Grafana (`http://:3000`, admin/bigfred) z datasource VictoriaMetrics
(`:8428`). Dane: `/data/opt/grafana`, `/data/opt/victoriametrics`.
Flush VM na dysk: `-inmemoryDataFlushInterval=5m` w `/etc/default/victoriametrics`.

Panel admina: `bigfred-os-ui` (`http://:8090`, konfiguracja w `/data/etc/bigfred-os-ui.conf`).

## Struktura

```text
os/
├── configs/           # defconfig, fragmenty jądra i BusyBox
├── board/bigfred_hub/ # cmdline, config.txt, genimage, post-*.sh
├── overlays/          # fstab, init.d, redis, crontab, udev
├── kernel/            # (fragmenty w configs/linux-hub.fragment)
├── package/           # alloy, grafana, victoriametrics (Go apps: ../apps/)
├── scripts/           # flash-nvme.sh
../apps/                 # Go apps → apps/.bin/ → /usr/sbin/ on image
├── Makefile
└── external.desc
```

## Dostosowanie

```bash
make menuconfig    # pakiety Buildroot
make image
```

Defconfig projektu: `configs/bigfred_hub_rpi5_defconfig`.
