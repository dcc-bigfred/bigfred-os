# configure-ethernet

Brings up the first Ethernet interface on the hub with a static address on common club subnets, or falls back to DHCP. Linux only.

Started at boot by `S15-network` (`/usr/sbin/configure-ethernet`).

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

No flags. Configuration is read from `/data/etc/configure-ethernet.conf` (created on first run if missing).

## Configuration

```ini
# configure-ethernet static addresses (edit to match club subnet)
PRIMARY=192.168.0.120
SECONDARY=192.168.1.120
```

| Key | Default | Description |
|-----|---------|-------------|
| `PRIMARY` | `192.168.0.120` | First static address to try (`PRIMARY_ADDRESS`, `ADDRESS` also accepted) |
| `SECONDARY` | `192.168.1.120` | Fallback static address (`SECONDARY_ADDRESS`, `FALLBACK`, `FALLBACK_ADDRESS` also accepted) |

Gateway is derived from the host address (last octet set to `.1`). Both static attempts use a `/24` prefix.

## Behaviour

1. Load or create `/data/etc/configure-ethernet.conf` (warn only if the file cannot be written).
2. Pick the first non-loopback, non-WiFi interface from `/sys/class/net`.
3. Try `PRIMARY` — configure the interface, ping the gateway; stop on success.
4. Try `SECONDARY` — same as above.
5. Run `dhclient` on the interface; success when an IPv4 address is assigned.

## Tests

```bash
make -C apps test
# or: make -C apps/configure-ethernet test
```
