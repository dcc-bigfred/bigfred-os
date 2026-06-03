# rotate-hub-logs

Rotates and prunes hub log files under `/data/logs` (see `docs/loconet-adapter` §8.9).

## Build (hub target)

```bash
make -C apps build
# or: make -C apps rotate-hub-logs
```

Produces `apps/.bin/rotate-hub-logs` (`linux/arm64`, static).

## Run on device

```bash
/usr/sbin/rotate-hub-logs
```

Flags: `-logroot`, `-retention-days`, `-max-bytes`, `-rotate-size` (defaults match the former shell script).

## Tests

```bash
make -C apps test
# or: make -C apps/rotate-hub-logs test
```
