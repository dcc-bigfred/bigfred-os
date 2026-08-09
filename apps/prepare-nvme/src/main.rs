//! prepare-nvme — migrate hub `/data` from microSD onto a local NVMe partition
//! and remount `/data` there so the SD can stay read-only (wear leveling).
//!
//! Partitioning uses [`gptman`] (pure Rust GPT). Format / mount / copy still
//! call BusyBox / e2fsprogs helpers (`mkfs.ext4`, `mount`, `cp`, …).

use std::fs;
use std::path::Path;
use std::time::Duration;

use prepare_nvme::*;
use prepare_nvme::logf;

fn main() {
    let cmd = std::env::args().nth(1).unwrap_or_else(|| "all".into());
    logf!("start command={cmd:?} pid={}", std::process::id());

    if cmd.as_str() == "-h" || cmd.as_str() == "--help" || cmd.as_str() == "help" {
        usage();
        return;
    }

    if !Path::new(BIGFRED_MARKER).is_dir() {
        eprintln!(
            "prepare-nvme: ERROR: {BIGFRED_MARKER} not found — refusing to run outside BigFred OS"
        );
        std::process::exit(1);
    }
    logf!("marker ok: {BIGFRED_MARKER}");

    let result = match cmd.as_str() {
        "all" | "run" => run_all(),
        "prepare" => stage_prepare(),
        "mount" => stage_mount(),
        other => {
            eprintln!("prepare-nvme: unknown command {other:?}");
            usage();
            std::process::exit(2);
        }
    };

    if let Err(e) = result {
        eprintln!("prepare-nvme: ERROR: {e}");
        log_mounts_relevant();
        std::process::exit(1);
    }
    logf!("done command={cmd:?} ok");
}

fn usage() {
    let prog = std::env::args().next().unwrap_or_else(|| "prepare-nvme".into());
    eprintln!("usage: {prog} [all|prepare|mount]");
    eprintln!("  all      prepare NVMe if needed, then mount as /data (default)");
    eprintln!("  prepare  create/format empty NVMe partition and copy /data onto it");
    eprintln!("  mount    if NVMe p1 is ext4, mount it on /data (unmount SD /data)");
}

fn run_all() -> Result<()> {
    stage_prepare().map_err(|e| Error::Msg(format!("prepare stage: {e}")))?;
    stage_mount().map_err(|e| Error::Msg(format!("mount stage: {e}")))?;
    Ok(())
}

// --- stage 1: prepare + copy -------------------------------------------------

fn stage_prepare() -> Result<()> {
    logf!("--- stage prepare ---");
    log_tools();
    log_mounts_relevant();

    let disk = find_nvme_disk()?;
    let Some(disk) = disk else {
        logf!("no NVMe disk under /sys/block/nvme* — skipping prepare");
        return Ok(());
    };
    logf!("using disk {disk}");
    log_disk_info(&disk);

    let (part, created) = ensure_partition(&disk)
        .map_err(|e| Error::Msg(format!("ensure partition on {disk}: {e}")))?;
    let Some(part) = part else {
        logf!("no usable partition on {disk} — skipping prepare");
        return Ok(());
    };
    logf!("target partition {part} (justCreated={created})");

    let fstype = blkid_type(&part)?;
    logf!("{part} blkid TYPE={fstype:?}");

    let mut need_format = created || fstype.is_empty();
    if need_format {
        logf!("format decision: needFormat=true (created={created} type={fstype:?})");
    }

    if !need_format && fstype == "ext4" {
        let empty = partition_looks_empty(&part)?;
        logf!("{part} emptiness check: empty={empty}");
        if empty {
            need_format = true;
            logf!("{part} is empty ext4 — will reformat");
        }
    } else if !need_format && fstype != "ext4" {
        let empty = partition_looks_empty(&part)
            .map_err(|e| Error::Msg(format!("inspect {part} ({fstype}): {e}")))?;
        logf!("{part} emptiness check (type={fstype:?}): empty={empty}");
        if empty {
            need_format = true;
            logf!("{part} has type {fstype:?} but is empty — will format ext4");
        } else {
            logf!("{part} has type {fstype:?} and is not empty — skipping prepare");
            return Ok(());
        }
    }

    if need_format {
        logf!("formatting {part} as ext4 (LABEL=bigfred-data)");
        format_ext4(&part).map_err(|e| Error::Msg(format!("format {part}: {e}")))?;
        if let Ok(t) = blkid_type(&part) {
            if !t.is_empty() {
                logf!("after format: blkid TYPE={t:?}");
            }
        }
    } else {
        logf!("{part} already prepared (ext4, not empty) — skipping copy");
        return Ok(());
    }

    copy_data_onto(&part)
}

fn ensure_partition(disk: &str) -> Result<(Option<String>, bool)> {
    let parts = list_partitions(disk)?;
    logf!("{disk} partitions: {parts:?}");
    if parts.len() == 1 {
        return Ok((Some(parts[0].clone()), false));
    }
    if parts.len() > 1 {
        logf!(
            "{disk} has {} partitions — using first ({})",
            parts.len(),
            parts[0]
        );
        return Ok((Some(parts[0].clone()), false));
    }

    logf!("no partitions on {disk} — creating one GPT partition via gptman");
    create_single_partition(disk)?;
    if let Err(e) = run_cmd(PARTPROBE_BIN, &[disk]) {
        logf!("warning: partprobe {disk}: {e} (continuing)");
    }
    let part = wait_for_partition(disk, Duration::from_secs(10))?;
    logf!("partition appeared: {part}");
    Ok((Some(part), true))
}

fn copy_data_onto(part: &str) -> Result<()> {
    fs::create_dir_all(TMP_MOUNT)?;
    if is_mounted(TMP_MOUNT) {
        logf!("{TMP_MOUNT} still mounted — umount first");
        if let Err(e) = run_cmd(UMOUNT_BIN, &[TMP_MOUNT]) {
            logf!("warning: umount {TMP_MOUNT}: {e}");
        }
    }
    logf!("mounting {part} on {TMP_MOUNT} for copy");
    run_cmd(MOUNT_BIN, &["-t", "ext4", part, TMP_MOUNT])
        .map_err(|e| Error::Msg(format!("mount {part} on {TMP_MOUNT}: {e}")))?;

    let umount_tmp = || {
        logf!("umount {TMP_MOUNT} after copy");
        if let Err(e) = run_cmd(UMOUNT_BIN, &[TMP_MOUNT]) {
            logf!("warning: umount {TMP_MOUNT}: {e}");
        }
    };

    let result = (|| -> Result<()> {
        if !is_mounted(DATA_MOUNT) {
            logf!("{DATA_MOUNT} is not mounted — nothing to copy");
            return Ok(());
        }
        let src_dev = mount_source(DATA_MOUNT);
        logf!(
            "copy source: {DATA_MOUNT} is {} (fstype from /proc/mounts)",
            describe_mount(DATA_MOUNT)
        );
        if !src_dev.is_empty() && same_device(&src_dev, part) {
            logf!("{DATA_MOUNT} is already on {part} — skip copy");
            return Ok(());
        }

        let src = format!("{DATA_MOUNT}/.");
        let dst = format!("{TMP_MOUNT}/");
        logf!("copying {src} → {dst} (cp -a, preserving permissions)");
        run_cmd(CP_BIN, &["-a", &src, &dst])
            .map_err(|e| Error::Msg(format!("copy {DATA_MOUNT} → {part}: {e}")))?;
        logf!("copy complete");

        // Marker so early-boot mounts NVMe directly on next boots.
        let marker = format!("{TMP_MOUNT}/{NVME_MARKER}");
        logf!("writing migration marker: {marker}");
        fs::write(&marker, b"bigfred-nvme\n")
            .map_err(|e| Error::Msg(format!("write marker: {e}")))?;
        Ok(())
    })();

    umount_tmp();
    result
}

// --- stage 2: mount NVMe as /data --------------------------------------------

fn stage_mount() -> Result<()> {
    logf!("--- stage mount ---");
    log_mounts_relevant();

    let disk = find_nvme_disk()?;
    let Some(disk) = disk else {
        logf!("no NVMe disk — skipping mount");
        return Ok(());
    };
    logf!("using disk {disk}");

    let parts = list_partitions(&disk)?;
    logf!("{disk} partitions: {parts:?}");
    if parts.is_empty() {
        logf!("no partition on {disk} — skipping mount");
        return Ok(());
    }
    let part = &parts[0];

    let fstype = blkid_type(part)?;
    logf!("{part} blkid TYPE={fstype:?}");
    if fstype != "ext4" {
        logf!("{part} type={fstype:?} (want ext4) — skipping mount");
        return Ok(());
    }

    if is_mounted(DATA_MOUNT) {
        let cur = mount_source(DATA_MOUNT);
        logf!("current {DATA_MOUNT} → {}", describe_mount(DATA_MOUNT));
        if same_device(&cur, part) {
            logf!("{part} already mounted on {DATA_MOUNT} — nothing to do");
            return Ok(());
        }
        log_busy_mount(DATA_MOUNT);
        logf!("unmounting {DATA_MOUNT} ({cur}) before switching to NVMe {part}");
        if let Err(e) = run_cmd(UMOUNT_BIN, &[DATA_MOUNT]) {
            log_busy_mount(DATA_MOUNT);
            return Err(Error::Msg(format!(
                "umount {DATA_MOUNT} (busy or in use?): {e}"
            )));
        }
        logf!("umount {DATA_MOUNT} ok");
    } else {
        logf!("{DATA_MOUNT} is not mounted");
    }

    fs::create_dir_all(DATA_MOUNT)?;
    logf!("mounting {part} on {DATA_MOUNT} (ext4,rw,noatime)");
    run_cmd(
        MOUNT_BIN,
        &["-t", "ext4", "-o", "rw,noatime", part, DATA_MOUNT],
    )
    .map_err(|e| Error::Msg(format!("mount {part} on {DATA_MOUNT}: {e}")))?;
    logf!("mount ok: {}", describe_mount(DATA_MOUNT));
    Ok(())
}
