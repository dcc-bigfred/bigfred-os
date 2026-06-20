# Grafana OSS

Prebuilt tarball from [dl.grafana.com](https://dl.grafana.com/oss/release/).
Architecture suffix is chosen from the Buildroot target CPU (`prebuilt-arch.mk`).

- Version: see `grafana.mk`
- Runtime data: `/data/opt/grafana`
- Config: `/etc/grafana/grafana.ini`
- VictoriaMetrics datasource: `/etc/grafana/provisioning/datasources/victoriametrics.yaml`

Enable with `BR2_PACKAGE_GRAFANA=y` in defconfig or menuconfig.
