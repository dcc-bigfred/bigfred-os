# Hub applications

Binaries for the hub image (`linux/arm64`, static). The top-level
`apps/Makefile` is **language-agnostic**: it discovers every
`apps/<name>/Makefile` and runs `make -C <name> build`.

## Build all

```bash
make -C apps build
```

Output: `apps/.bin/<app-name>` (gitignored). Exception: `bigfred-os-ui`
writes `bigfred-os-ui-linux-arm64` (and a host binary for local use).

## Build one app

```bash
make -C apps rotate-hub-logs
# or:
make -C apps/prepare-nvme build
```

## Test / clean

```bash
make -C apps test
make -C apps clean
```

## Apps

| App | Lang | Role |
|-----|------|------|
| `bigfred-os-ui` | Go | Hub admin web UI (logs, services) |
| `rotate-hub-logs` | Go | Log rotation under `/data/logs` |
| `fanctl` | Go | Pi 5 active cooler (§8.8) |
| `configure-ethernet` | Go | One-shot Ethernet `up` + `check` for microinit liveness |
| `prepare-nvme` | Rust | Migrate `/data` to NVMe (GPT via gptman) and remount |

Init/PID 1 is **microinit** (OCI package), not under `apps/`. The SysV-style
`shutdown` CLI ships with that same OCI bundle (`/usr/sbin/shutdown`).

## Adding an app

1. Create `apps/<name>/` with a `Makefile` that implements:
   - `build` — write the binary into `$(BIN_DIR)/…` (`BIN_DIR` defaults to `apps/.bin`)
   - `test` — unit/integration tests
   - `clean` — remove that app’s artifacts under `BIN_DIR`
2. `make -C apps build` picks it up automatically (any dir with a Makefile).
3. `os/board/bigfred_hub/post-build.sh` installs executables from `apps/.bin/` into `/usr/sbin/`.
