# Hub applications

Go binaries built for the hub image (`linux/arm64`, static).

## Build all

```bash
make -C apps build
```

Output: `apps/.bin/<app-name>` (gitignored).

## Build one app

```bash
make -C apps rotate-hub-logs
```

## Test

```bash
make -C apps test
```

## Apps

| App | Role |
|-----|------|
| `rotate-hub-logs` | Log rotation under `/data/logs` |
| `fanctl` | Pi 5 active cooler (§8.8) |

## Adding an app

1. Create `apps/<name>/main.go`
2. `make -C apps build` — the new binary appears in `apps/.bin/<name>`
3. `os/board/bigfred_hub/post-build.sh` installs every executable from `apps/.bin/` into `/usr/sbin/` on the image
