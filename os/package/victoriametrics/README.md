# VictoriaMetrics

Prebuilt single-node binary (statically linked). Architecture suffix is
chosen from the Buildroot target CPU (`prebuilt-arch.mk`).

- Version: see `victoriametrics.mk`
- Storage at runtime: `/data/opt/victoriametrics`
- Flush interval: `-inmemoryDataFlushInterval` in `/etc/default/victoriametrics`
  (default `5m` — less frequent disk writes than the upstream 5s default)

Enable with `BR2_PACKAGE_VICTORIAMETRICS=y` in defconfig or menuconfig.
