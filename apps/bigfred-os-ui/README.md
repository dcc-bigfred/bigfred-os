# BigFred Hub OS — admin web UI

Frontend to manage and diagnose BigFred OS based on Linux.

## Build

```bash
make -C apps/bigfred-os-ui build
Produces `apps/.bin/bigfred-os-ui-linux-<arch>` (hub, PAM) and
`apps/.bin/bigfred-os-ui-<host-os>-<host-arch>` (local dev, static auth).
```

Requires Node.js for the frontend bundle (`web/dist` embedded via `go:embed`).
All fonts and scripts are vendored at build time — no CDN at runtime
(see [§7b offline assets](https://dcc-bigfred.github.io/docs/bigfred/architecture/09b-offline-assets/)).

## Run

```bash
./apps/.bin/bigfred-os-ui \
  --config /data/etc/bigfred-os-ui.conf
```

Or pass flags directly (override config file values). On the hub image login uses **PAM**
(`root` / password from `/data/etc/shadow`). For local dev without PAM:

```bash
go run -tags '!pam' ./apps/bigfred-os-ui \
  --http 0.0.0.0:8090 \
  --username root \
  --password 'root' \
  --log-roots /data/logs,/var/log
```

Hub production binary is built with `-tags pam` and links `libpam` (see `Makefile`).

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `/data/etc/bigfred-os-ui.conf` | Dotenv file (`KEY=value`) |
| `--http` | `0.0.0.0:8090` | Listen address |
| `--pam-service` | `bigfred-os-ui` | PAM service name (`/etc/pam.d/…`) |
| `--username` | *(PAM)* | Static login (dev, `-tags '!pam'` only) |
| `--password` | *(PAM)* | Static password (dev, `-tags '!pam'` only) |
| `--log-roots` | `/data/logs,/var/log` | Comma-separated log directories |
| `--log-root` | *(deprecated)* | Single log directory |
| `--init-dir` | `/etc/init.d` | SysV init scripts directory |
| `--supervisord-conf` | `/data/etc/supervisord/supervisord.conf` | supervisord configuration file |
| `--update-dir` | `/data/opt/bigfred/bin` | Install dir for Update tab downloads |
| `--github-token` | *(env `GITHUB_TOKEN`)* | Token for private GitHub release downloads |
| `--secure-cookie` | `false` | Set `Secure` on session cookie (HTTPS) |
| `--static-dir` | *(embedded)* | Serve frontend from disk (dev) |

CLI flags override values from `--config`. On the hub image, `S48-bigfred-os-ui`
starts the binary with `--config /data/etc/bigfred-os-ui.conf`.

### Dotenv format (`/data/etc/bigfred-os-ui.conf`)

```dotenv
HTTP=0.0.0.0:8090
PAM_SERVICE=bigfred-os-ui
LOG_ROOTS=/data/logs,/var/log
SECURE_COOKIE=false
# Optional: UPDATE_DIR=/data/opt/bigfred/bin
# Optional: GITHUB_TOKEN=…
```

### Update tab

Authenticated operators can download the latest GitHub release assets into
`/data/opt/bigfred/bin`:

| Button | Repo | Asset (arm64) | Installed as |
|--------|------|---------------|--------------|
| Update BigFred | `dcc-bigfred/bigfred` | `loco-server-linux-arm64` | `bigfred` |
| Update BigFred UI | `dcc-bigfred/bigfred-os` | `bigfred-os-ui-linux-arm64` | `bigfred-os-ui` |

After install, restart the matching SysV service from **Services**
(`bigfred` / `bigfred-os-ui`). `S48-bigfred-os-ui` and `/usr/bin/bigfred`
prefer the `/data/opt/bigfred/bin` copies.

Seed template ships as `/etc/bigfred/bigfred-os-ui.conf` and is copied to
`/data/etc/` on first boot (see `S10-mount`).

## Development

Terminal 1 — backend with live static files (after `npm run build` once, or use Vite):

```bash
make -C apps/bigfred-os-ui dev-backend USERNAME=admin PASSWORD=admin
```

Terminal 2 — Vite dev server (proxies `/api` to :8090):

```bash
make -C apps/bigfred-os-ui dev-web
```

Open http://localhost:5174

## Tabs

| Tab | Status |
|-----|--------|
| **Logs** | Live tail over WebSocket (`/api/v1/logs/stream`) |
| **Terminal** | Interactive shell over WebSocket (`/api/v1/terminal`, PTY + xterm.js; requires login) |
| **Supervisord** | Programs from `/data/etc/supervisord/supervisord.conf` — start/stop/restart via `supervisorctl` |
| **Services** | SysV init scripts from `/etc/init.d` — start/stop/restart |
| **Update** | Download latest GitHub release binaries into `/data/opt/bigfred/bin` |

## API

- `POST /api/v1/auth/login` — session cookie (JWT, PAM on hub)
- `GET /api/v1/auth/me` — current user
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/password` — change Linux password (PAM)
- `GET /api/v1/services` — list init scripts and running state
- `POST /api/v1/services/{id}/{action}` — `start`, `stop`, or `restart`
- `GET /api/v1/supervisord/programs` — list supervisord programs (config + status)
- `POST /api/v1/supervisord/programs/{name}/{action}` — `start`, `stop`, or `restart`
- `POST /api/v1/update/{target}` — `bigfred` or `bigfred-ui` with body `{"tag":"v1.2.3"}` → `/data/opt/bigfred/bin`
- `GET /api/v1/update/{target}/releases` — list GitHub releases that include the target asset
- `GET /api/v1/logs` — list log files from configured roots
- `GET /api/v1/logs/stream?id=<root-id:path>` — WebSocket stream
- `GET /api/v1/terminal` — WebSocket interactive shell (PTY)
