# configure-ethernet

Long-running Ethernet bring-up daemon for the hub. Tries common club static
subnets, falls back to DHCP, then stays up and re-runs the same bring-up when
the connection is lost. Linux only.

Started at boot by `S15-network` / microinit service `network`
(`/usr/sbin/configure-ethernet`).

## Build (hub target)

```bash
make -C apps build
# or: make -C apps configure-ethernet
```

Produces `apps/.bin/configure-ethernet` (`linux/arm64`, static).

## Run on device

```bash
/usr/sbin/configure-ethernet
```

Runs in the foreground until SIGTERM/SIGINT. Configuration is read from
`/data/etc/configure-ethernet.conf` (created on first run if missing).

## Configuration

```ini
# configure-ethernet static addresses (edit to match club subnet)
PRIMARY=192.168.0.120
SECONDARY=192.168.1.120
# Seconds between health checks while connected
CONNECTION_TIMEOUT=10
# Seconds to wait after a failed bring-up before retrying
BACKOFF_TIME=30
```

| Key | Default | Description |
|-----|---------|-------------|
| `PRIMARY` | `192.168.0.120` | First static address to try (`PRIMARY_ADDRESS`, `ADDRESS` also accepted) |
| `SECONDARY` | `192.168.1.120` | Fallback static address (`SECONDARY_ADDRESS`, `FALLBACK`, `FALLBACK_ADDRESS` also accepted) |
| `CONNECTION_TIMEOUT` | `10` | Seconds between health checks while the link is up |
| `BACKOFF_TIME` | `30` | Seconds to wait after a failed bring-up before the next attempt |

Gateway is derived from the host address (last octet set to `.1`). Both static attempts use a `/24` prefix.

## Behaviour

1. Load or create `/data/etc/configure-ethernet.conf`.
2. Pick the first non-loopback, non-WiFi interface from `/sys/class/net`.
3. Try `PRIMARY` → `SECONDARY` → `dhclient` (same bring-up as before).
4. On failure: wait `BACKOFF_TIME` and retry step 2–3.
5. On success: every `CONNECTION_TIMEOUT` seconds check link/IPv4/(gateway ping).
6. If the check fails: log loss and immediately re-run the bring-up (step 2–3).

## Tests

```bash
make -C apps test
# or: make -C apps/configure-ethernet test
```
