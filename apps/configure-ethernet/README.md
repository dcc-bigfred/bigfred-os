# configure-ethernet

One-shot Ethernet bring-up for the hub. Tries common club static subnets, then
falls back to DHCP. A separate `check` subcommand is a cheap liveness probe for
microinit. Linux only.

Started at boot by `S15-network` / microinit service `network`
(`/usr/sbin/configure-ethernet`).

## Build (hub target)

```bash
make -C apps build
# or: make -C apps configure-ethernet
```

Produces `apps/.bin/configure-ethernet` (`linux/arm64`, static).

## Commands

```bash
/usr/sbin/configure-ethernet          # same as "up"
/usr/sbin/configure-ethernet up       # configure once and exit
/usr/sbin/configure-ethernet check    # exit 0 if UP+IPv4, else 1
```

`check` only looks at link state and IPv4 on the first Ethernet interface (no ping).

Configuration is read from `/data/etc/configure-ethernet.conf` (created on first run if missing).

## Configuration

```ini
# configure-ethernet static addresses (edit to match club subnet)
PRIMARY=192.168.0.120
SECONDARY=192.168.1.120
```

| Key | Default | Description |
|-----|---------|-------------|
| `PRIMARY` | `192.168.0.120` | First static address to try |
| `SECONDARY` | `192.168.1.120` | Fallback static address |

Gateway is derived from the host address (last octet set to `.1`). Both static attempts use a `/24` prefix.

## microinit liveness

On the hub, `network` is a one-shot job with a liveness probe:

```json
"livenessProbe": {
  "cmd": "/usr/sbin/configure-ethernet check",
  "successExitCodes": [0],
  "interval": 30
}
```

When `check` fails, microinit re-runs the network start (bring-up again).

## Tests

```bash
make -C apps test
# or: make -C apps/configure-ethernet test
```
