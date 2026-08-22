# Kernel configuration

Kernel options for the hub are applied via Buildroot fragment files:

- `../configs/linux-4k-page-size.fragment` — 4K pages (Pi 5 / aarch64; 16K/64K off)
- `../configs/linux-hub.fragment` — PREEMPT, watchdog, USB-ACM, ext4, brcmfmac

The kernel source is fetched by Buildroot from `raspberrypi/linux` **rpi-6.18.y**,
pinned by commit SHA in `configs/bigfred_hub_rpi5_defconfig`
(`BR2_LINUX_KERNEL_CUSTOM_TARBALL_*`). The tarball hash lives in
`board/bigfred_hub/patches/linux/linux.hash`. To move the pin, update the SHA,
recompute the hash, and `linux-dirclean` before rebuilding.
