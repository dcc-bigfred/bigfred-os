# Grafana OSS

Prebuilt tarball from [dl.grafana.com](https://dl.grafana.com/oss/release/).
Architecture suffix is chosen from the Buildroot target CPU (`prebuilt-arch.mk`).

- Version: see `grafana.mk`
- Binary: `/usr/bin/grafana` (`grafana server` / `grafana cli`)
- Runtime data: `/data/var/lib/grafana`
- Config: `/etc/grafana/grafana.ini`
- VictoriaMetrics datasource: `/etc/grafana/provisioning/datasources/victoriametrics.yaml`
- Dashboard provisioning: `/etc/grafana/provisioning/dashboards/dashboards.yaml`
- Runtime dashboards: `/data/etc/grafana/dashboards/` (seeded from `/usr/share/grafana/dashboards/` at boot)

At image build time, `fetch-dashboards.sh` pulls `misc/grafana/dashboards/*.json`
from GitHub using `BIGFRED_REF` and `MICROINIT_REF` (repo-root `VERSIONS`).
Each `make image` re-fetches dashboards without re-downloading the OSS tarball.

Enable with `BR2_PACKAGE_GRAFANA=y` in defconfig or menuconfig.
