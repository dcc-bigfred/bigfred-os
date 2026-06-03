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

Optional sysfs overrides: `FANCTL_FAN_PWM`, `FANCTL_FAN_MAX`, `FANCTL_THERM`.
