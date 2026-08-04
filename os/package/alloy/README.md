# Grafana Alloy

Built from source ([grafana/alloy](https://github.com/grafana/alloy)) as a
**static** `linux/<arch>` binary (`CGO_ENABLED=0`, `netgo`/`osusergo`).

The official GitHub release zip is dynamically linked against glibc and does
not run on the hub's musl rootfs (`alloy: not found` despite the file existing).

Version is pinned in `alloy.mk`. Requires Go ≥ 1.26 on the build host
(`docker/install-go.sh`; `GOTOOLCHAIN=auto` may download the exact toolchain
named in Alloy's `go.mod`).

Enable with `BR2_PACKAGE_ALLOY=y` in defconfig or menuconfig.
