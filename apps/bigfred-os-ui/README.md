# BigFred Hub OS — admin web UI

Frontend to manage and diagnose BigFred OS based on Linux.

## Build

```bash
make -C apps/bigfred-os-ui build
# → apps/.bin/bigfred-os-ui
```

Requires Node.js for the frontend bundle (`web/dist` embedded via `go:embed`).
All fonts and scripts are vendored at build time — no CDN at runtime
(see [§7b offline assets](https://dcc-bigfred.github.io/docs/bigfred/architecture/09b-offline-assets/)).

## Run

```bash
./apps/.bin/bigfred-os-ui \
  --config /data/etc/bigfred-os-ui.conf
```

Or pass flags directly (override config file values):

```bash
./apps/.bin/bigfred-os-ui \
  --http 0.0.0.0:8090 \
  --username admin \
  --password 'change-me' \
  --log-roots /data/logs,/var/log
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `/data/etc/bigfred-os-ui.conf` | Dotenv file (`KEY=value`) |
| `--http` | `0.0.0.0:8090` | Listen address |
| `--username` | *(required)* | Login |
| `--password` | *(required)* | Password |
| `--log-roots` | `/data/logs,/var/log` | Comma-separated log directories |
| `--log-root` | *(deprecated)* | Single log directory |
| `--secure-cookie` | `false` | Set `Secure` on session cookie (HTTPS) |
| `--static-dir` | *(embedded)* | Serve frontend from disk (dev) |

CLI flags override values from `--config`. On the hub image, `S48-bigfred-os-ui`
starts the binary with `--config /data/etc/bigfred-os-ui.conf`.

### Dotenv format (`/data/etc/bigfred-os-ui.conf`)

```dotenv
HTTP=0.0.0.0:8090
USERNAME=admin
PASSWORD=bigfred
LOG_ROOTS=/data/logs,/var/log
SECURE_COOKIE=false
```

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
| Supervisord | Placeholder |
| **Services** | SysV init scripts from `/etc/init.d` — start/stop/restart |

## API

- `POST /api/v1/auth/login` — session cookie (JWT)
- `GET /api/v1/auth/me` — current user
- `POST /api/v1/auth/logout`
- `GET /api/v1/services` — list init scripts and running state
- `POST /api/v1/services/{id}/{action}` — `start`, `stop`, or `restart`
- `GET /api/v1/logs` — list log files from configured roots
- `GET /api/v1/logs/stream?id=<root-id:path>` — WebSocket stream
