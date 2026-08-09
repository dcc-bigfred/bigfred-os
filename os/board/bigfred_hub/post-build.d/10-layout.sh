# Mount point for RW data partition (ext4 LABEL=bigfred-data on SD or NVMe)
mkdir -p "${TARGET_DIR}/data"

# Persistent log and application state directories (on /data at runtime).
# Modes here are placeholders; early-boot.sh applies 0750 + service chown
# after users.table accounts exist (makeusers runs after post-build).
mkdir -p "${TARGET_DIR}/data/var/db/redis"
mkdir -p "${TARGET_DIR}/data/var/db/bigfred"
mkdir -p "${TARGET_DIR}/data/var/lib/alloy"
mkdir -p "${TARGET_DIR}/data/var/lib/victoriametrics"
mkdir -p "${TARGET_DIR}/data/var/lib/grafana/data"
mkdir -p "${TARGET_DIR}/data/var/lib/grafana/plugins"
# bigfred $HOME (tmpfs mounted at early-boot on RO root)
mkdir -p "${TARGET_DIR}/home/bigfred"
# Override path for BigFred binary (/usr/bin/bigfred prefers this over /opt)
mkdir -p "${TARGET_DIR}/data/opt/bigfred/bin"
mkdir -p "${TARGET_DIR}/data/logs/bigfred"
mkdir -p "${TARGET_DIR}/data/logs/redis"
mkdir -p "${TARGET_DIR}/data/logs/alloy"
mkdir -p "${TARGET_DIR}/data/logs/grafana"
mkdir -p "${TARGET_DIR}/data/etc"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/infra"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/dcc-bus"
mkdir -p "${TARGET_DIR}/data/etc/microinit.d/services/os"
# Persistent root SSH (bound over /root/.ssh at early-boot)
mkdir -p "${TARGET_DIR}/data/root/.ssh"
chmod 700 "${TARGET_DIR}/data/root/.ssh"
# Mount point on RO rootfs for the .ssh bind
mkdir -p "${TARGET_DIR}/root/.ssh"
chmod 700 "${TARGET_DIR}/root/.ssh"

# Placeholder for BigFred (installed separately by operator)
mkdir -p "${TARGET_DIR}/usr/share/bigfred/web"
