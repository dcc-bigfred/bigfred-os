# OS-owned microinit services

Each `*.json` file is a microinit drop-in with a single service:

```json
{ "services": [ { "name": "...", ... } ] }
```

At boot, `early-boot.sh` always copies this directory from the image into
`$DATA_DIR/etc/microinit.d/services/os/` so upgrades pick up new services
(for example `microdns`) without rewriting an existing main
`microinit.json`.

Do not put loco-server-managed services here — those live under `infra/`
and `dcc-bus/` and are written by BigFred at runtime.
