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

## `orderPriority`

Lower value = earlier among services that are currently ready
(`dependsOn` satisfied). Equal values fall back to alphabetical name.
`dependsOn` is a hard DAG edge and always wins over priority.

Bands used on the hub (leave gaps so new services fit between):

| Band | Range | Purpose |
|------|-------|---------|
| Host prep | 10–29 | sysctl, udevd, watchdog |
| Network | 30–49 | network |
| Housekeeping | 50–99 | cron |
| Infra | 100–149 | redis, microdns, victoriametrics, alloy, microwaf |
| Access / ops | 200–249 | dropbear, remote-icmp, bigfred-os-ui |
| Apps | 300–349 | bigfred |
| Observability UI | 400+ | grafana |

| Service | orderPriority | dependsOn |
|---------|---------------|-----------|
| sysctl | 10 | — |
| udevd | 15 | — |
| watchdog | 20 | — |
| network | 30 | — |
| cron | 50 | — |
| redis | 100 | network |
| microdns | 110 | network |
| victoriametrics | 120 | network |
| alloy | 130 | network |
| microwaf | 140 | network, redis (installed, **disabled** at boot) |
| dropbear | 200 | network |
| remote-icmp | 210 | network |
| bigfred-os-ui | 220 | network |
| bigfred | 300 | network, redis, udevd |
| bigfred-wizard | 310 | network, bigfred |
| grafana | 400 | victoriametrics |

Runtime `dcc-bus-*` drop-ins (written by BigFred) use **350** on the same scale.
