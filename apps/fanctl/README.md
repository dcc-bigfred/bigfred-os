# fanctl

Raspberry Pi 5 active cooler control (hub spec §8.8).

| SoC temperature | Fan level |
|-----------------|-----------|
| below 45 °C     | OFF (0)   |
| 45–60 °C        | LOW (1)   |
| 60–70 °C        | MED (2)   |
| above 70 °C     | HIGH (3)  |

## Build

```bash
make -C apps build
```

Installed to `/usr/sbin/fanctl` on the hub image (`S50-fanctl`).

## Usage

```bash
fanctl daemon   # foreground loop (init runs via start-stop-daemon)
fanctl stop     # fan off
```

Configuration is read from `/data/etc/fanctl.conf` (created on first run if missing).
On first boot the image seeds the file from `/etc/bigfred/fanctl.conf` via `S10-mount`.

### Config format (`/data/etc/fanctl.conf`)

```ini
COOLDOWN=300
INTERVAL=5
```

| Key | Default | Description |
|-----|---------|-------------|
| `COOLDOWN` | `300` | Minimum seconds the fan stays on after turn-on before it may be switched off (`0` disables) |
| `INTERVAL` | `5` | Temperature polling interval in seconds |
| `FAN_PWM` | `.../cooling_device0/cur_state` | Fan PWM sysfs path |
| `FAN_MAX` | `.../cooling_device0/max_state` | Fan max state sysfs path |
| `THERM` | `.../thermal_zone0/temp` | Temperature sysfs path |

Environment variables (`FANCTL_COOLDOWN`, `FANCTL_INTERVAL`, `FANCTL_FAN_PWM`, …) override the config file when set.
