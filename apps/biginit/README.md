# biginit

SysV boot orchestrator for the hub (`/etc/init.d/rcS` replacement).

Runs all executable `S??-*` init scripts in lexical order, matching BusyBox `rcS` behaviour for regular scripts (`script start`) and sourced `.sh` scripts.

## Configuration

After `S10-mount` mounts `/data`, biginit loads `/data/etc/biginit.yaml`. If the file is missing, it is created from discovered services. `/data/etc/biginit.yaml.defaults` is always written with built-in defaults.

```yaml
services:
  - name: dropbear
    autostart: false
    retries: 3
```

| Field | Meaning |
|-------|---------|
| `name` | Service id from init script (`S90-dropbear` → `dropbear`) |
| `autostart` | Start during system boot |
| `retries` | Extra attempts after the first failed start |

Services not listed in the config default to `autostart: true`, `retries: 0`.

`S10-mount` always runs before the config is loaded; all other services honour the config.

## Build

```bash
make -C apps biginit
# or: make -C apps test
```

Installed to `/usr/sbin/biginit`; `rcS` execs it at sysinit.
