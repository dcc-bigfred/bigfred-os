//! Shared NVMe disk helpers for BigFred OS tooling (`prepare-nvme`, factory-reset, …).
//!
//! Logging uses [`logf!`], which prints lines prefixed with `nvme:`.

use std::fs::{self, File, OpenOptions};
use std::io::{self, Write};
use std::os::unix::fs::FileTypeExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use gptman::{GPTPartitionEntry, GPT};
use thiserror::Error;

/// Emit a log line to stderr with the shared `nvme:` prefix.
pub fn emit(args: std::fmt::Arguments<'_>) {
    let _ = writeln!(std::io::stderr(), "nvme: {}", args);
}

/// Log a line prefixed with `nvme:`.
#[macro_export]
macro_rules! logf {
    ($($t:tt)*) => {
        $crate::emit(format_args!($($t)*))
    };
}

pub const DATA_MOUNT: &str = "/data";
pub const TMP_MOUNT: &str = "/run/prepare-nvme";
/// Marker file written at the root of the NVMe data partition after a
/// successful migration. early-boot.sh looks for it to mount NVMe directly
/// on subsequent boots (bypassing the microSD data partition).
pub const NVME_MARKER: &str = ".bigfred-nvme";
/// Marker directory shipped on BigFred OS images (see post-build.sh).
pub const BIGFRED_MARKER: &str = "/var/lib/bigfred";

pub const MKE2FS_BIN: &str = "/sbin/mke2fs";
pub const MKFS_EXT4_BIN: &str = "/sbin/mkfs.ext4";
pub const BLKID_BIN: &str = "/sbin/blkid";
pub const PARTPROBE_BIN: &str = "/usr/sbin/partprobe";
pub const MOUNT_BIN: &str = "/bin/mount";
pub const UMOUNT_BIN: &str = "/bin/umount";
pub const CP_BIN: &str = "/bin/cp";
pub const FUSER_BIN: &str = "/bin/fuser";

/// Linux filesystem partition type GUID (GPT mixed-endian encoding of
/// `0FC63DAF-8483-4772-8E79-3D69D8477DE4`).
pub const LINUX_FS_GUID: [u8; 16] = [
    0xaf, 0x3d, 0xc6, 0x0f, 0x83, 0x84, 0x72, 0x47, 0x8e, 0x79, 0x3d, 0x69, 0xd8, 0x47, 0x7d, 0xe4,
];

#[derive(Debug, Error)]
pub enum Error {
    #[error("{0}")]
    Msg(String),
    #[error(transparent)]
    Io(#[from] io::Error),
}

pub type Result<T> = std::result::Result<T, Error>;

/// Unmount every mount whose device path starts with `disk` (e.g. `/dev/nvme0n1`).
/// Best-effort; returns first hard error if umount fails for a matching mount.
pub fn unmount_disk_mounts(disk: &str) -> Result<()> {
    let Ok(content) = fs::read_to_string("/proc/mounts") else {
        logf!("cannot read /proc/mounts — skipping unmount_disk_mounts");
        return Ok(());
    };

    let part_prefix = format!("{disk}p");
    let mut mountpoints: Vec<String> = Vec::new();
    for line in content.lines() {
        let mut fields = line.split_whitespace();
        let Some(dev) = fields.next() else { continue };
        let Some(mp) = fields.next() else { continue };
        if dev == disk || dev.starts_with(&part_prefix) {
            if !mountpoints.iter().any(|m| m == mp) {
                mountpoints.push(mp.to_string());
            }
        }
    }

    // Longest mountpoint first so nested mounts are torn down before parents.
    mountpoints.sort_by_key(|m| std::cmp::Reverse(m.len()));

    for mp in &mountpoints {
        logf!("umount {mp} (device under {disk})");
        run_cmd(UMOUNT_BIN, &[mp]).map_err(|e| {
            Error::Msg(format!("umount {mp} (disk {disk}): {e}"))
        })?;
    }
    Ok(())
}

/// Force a fresh single GPT partition on `disk` (destroys existing table), wait
/// for it, format ext4. Does NOT copy data and does NOT write the `.bigfred-nvme`
/// marker.
pub fn wipe_format_data_partition(disk: &str) -> Result<String> {
    logf!("wipe_format_data_partition: {disk}");
    create_single_partition(disk)?;
    // partprobe / reread already attempted inside create_single_partition
    let part = wait_for_partition(disk, Duration::from_secs(10))?;
    logf!("wipe: formatting {part} as ext4 (LABEL=bigfred-data)");
    format_ext4(&part)?;
    Ok(part)
}

pub fn find_nvme_disk() -> Result<Option<String>> {
    let entries = match fs::read_dir("/sys/block") {
        Ok(e) => e,
        Err(e) if e.kind() == io::ErrorKind::NotFound => {
            logf!("/sys/block missing — cannot discover NVMe");
            return Ok(None);
        }
        Err(e) => return Err(e.into()),
    };

    let mut disks = Vec::new();
    let mut skipped = Vec::new();
    for ent in entries {
        let ent = ent?;
        let name = ent.file_name().to_string_lossy().into_owned();
        if !name.starts_with("nvme") {
            continue;
        }
        // Whole-disk names look like nvme0n1 (partitions are not under /sys/block).
        if !name.contains('n') {
            skipped.push(format!("{name}(no-n)"));
            continue;
        }
        let dev = format!("/dev/{name}");
        match fs::metadata(&dev) {
            Ok(st) if st.file_type().is_block_device() => disks.push(dev),
            Ok(_) => skipped.push(format!("{name}(not-block)")),
            Err(e) => skipped.push(format!("{name}(stat:{e})")),
        }
    }
    if !skipped.is_empty() {
        logf!("NVMe sysfs candidates skipped: {skipped:?}");
    }
    logf!("NVMe disks found: {disks:?}");
    if disks.is_empty() {
        return Ok(None);
    }
    for d in &disks {
        if Path::new(d).file_name().and_then(|s| s.to_str()) == Some("nvme0n1") {
            return Ok(Some(d.clone()));
        }
    }
    Ok(Some(disks[0].clone()))
}

pub fn list_partitions(disk: &str) -> Result<Vec<String>> {
    let base = Path::new(disk)
        .file_name()
        .and_then(|s| s.to_str())
        .unwrap_or(disk);
    let pattern = format!("/dev/{base}p*");
    let mut out = Vec::new();
    // Manual glob via read_dir of /dev — avoid extra crate; match prefix.
    let prefix = format!("/dev/{base}p");
    if let Ok(entries) = fs::read_dir("/dev") {
        for ent in entries.flatten() {
            let p = ent.path();
            let s = p.to_string_lossy();
            if !s.starts_with(&prefix) {
                continue;
            }
            let rest = &s[prefix.len()..];
            if rest.is_empty() || !rest.chars().all(|c| c.is_ascii_digit()) {
                continue;
            }
            if let Ok(st) = fs::metadata(&p) {
                if st.file_type().is_block_device() {
                    out.push(s.into_owned());
                }
            }
        }
    }
    // Also try glob via Path if pattern matched nothing (fallback).
    if out.is_empty() {
        // Keep empty — no partitions.
        let _ = pattern;
    }
    sort_partitions(&mut out, base);
    Ok(out)
}

pub fn sort_partitions(parts: &mut [String], disk_base: &str) {
    let prefix = format!("/dev/{disk_base}p");
    parts.sort_by_key(|p| partition_index(p, &prefix));
}

pub fn partition_index(path: &str, prefix: &str) -> u32 {
    let s = path.strip_prefix(prefix).unwrap_or(path);
    let mut n = 0u32;
    for c in s.chars() {
        if !c.is_ascii_digit() {
            break;
        }
        n = n
            .saturating_mul(10)
            .saturating_add(u32::from(c.to_digit(10).unwrap_or(0)));
    }
    n
}

pub fn wait_for_partition(disk: &str, timeout: Duration) -> Result<String> {
    let deadline = Instant::now() + timeout;
    let mut attempt = 0u32;
    while Instant::now() < deadline {
        attempt += 1;
        let parts = list_partitions(disk)?;
        if !parts.is_empty() {
            logf!("waitForPartition: appeared after {attempt} poll(s): {parts:?}");
            return Ok(parts[0].clone());
        }
        let base = Path::new(disk)
            .file_name()
            .and_then(|s| s.to_str())
            .unwrap_or("?");
        if attempt == 1 || attempt % 5 == 0 {
            logf!("waitForPartition: still no {base}p* (attempt {attempt})");
        }
        if attempt == 1 {
            if let Err(e) = run_cmd(PARTPROBE_BIN, &[disk]) {
                logf!("warning: partprobe during wait: {e}");
            }
        }
        thread::sleep(Duration::from_millis(200));
    }
    Err(Error::Msg(format!(
        "timed out waiting for partition on {disk} after {timeout:?} ({attempt} polls)"
    )))
}

pub fn create_single_partition(disk: &str) -> Result<()> {
    logf!("gptman: open {disk} read/write");
    let mut f = OpenOptions::new()
        .read(true)
        .write(true)
        .open(disk)
        .map_err(|e| Error::Msg(format!("open {disk}: {e}")))?;

    let sector_size = match gptman::linux::get_sector_size(&mut f) {
        Ok(s) => {
            logf!("gptman: sector size from ioctl = {s}");
            s as u64
        }
        Err(e) => {
            logf!("warning: get_sector_size failed ({e}) — assuming 512");
            512
        }
    };

    let disk_guid = random_guid();
    logf!(
        "gptman: writing protective MBR + new GPT (sector_size={sector_size}, disk_guid={})",
        guid_hex(&disk_guid)
    );

    GPT::write_protective_mbr_into(&mut f, sector_size)
        .map_err(|e| Error::Msg(format!("write protective MBR: {e}")))?;

    let mut gpt = GPT::new_from(&mut f, sector_size, disk_guid)
        .map_err(|e| Error::Msg(format!("GPT::new_from: {e}")))?;

    let part_guid = random_guid();
    gpt[1] = GPTPartitionEntry {
        partition_type_guid: LINUX_FS_GUID,
        unique_partition_guid: part_guid,
        starting_lba: gpt.header.first_usable_lba,
        ending_lba: gpt.header.last_usable_lba,
        attribute_bits: 0,
        partition_name: "bigfred-data".into(),
    };
    logf!(
        "gptman: partition #1 Linux FS LBA {}..{} name=bigfred-data guid={}",
        gpt.header.first_usable_lba,
        gpt.header.last_usable_lba,
        guid_hex(&part_guid)
    );

    gpt.write_into(&mut f)
        .map_err(|e| Error::Msg(format!("GPT::write_into: {e}")))?;
    f.sync_all()
        .map_err(|e| Error::Msg(format!("sync {disk}: {e}")))?;
    logf!("gptman: GPT written and synced");

    match gptman::linux::reread_partition_table(&mut f) {
        Ok(()) => logf!("gptman: reread_partition_table ioctl ok"),
        Err(e) => {
            logf!("warning: reread_partition_table: {e}; falling back to partprobe");
            if let Err(e2) = run_cmd(PARTPROBE_BIN, &[disk]) {
                logf!("warning: partprobe {disk}: {e2}");
            }
        }
    }
    Ok(())
}

pub fn format_ext4(part: &str) -> Result<()> {
    let label = "bigfred-data";
    if Path::new(MKFS_EXT4_BIN).exists() {
        logf!("using {MKFS_EXT4_BIN}");
        return run_cmd(MKFS_EXT4_BIN, &["-F", "-L", label, part]);
    }
    logf!("{MKFS_EXT4_BIN} missing — falling back to {MKE2FS_BIN}");
    run_cmd(MKE2FS_BIN, &["-t", "ext4", "-F", "-L", label, part])
}

pub fn blkid_type(dev: &str) -> Result<String> {
    let output = Command::new(BLKID_BIN)
        .args(["-o", "value", "-s", "TYPE", dev])
        .output()?;
    let out = String::from_utf8_lossy(&output.stdout).trim().to_string();
    let err_out = String::from_utf8_lossy(&output.stderr).trim().to_string();
    if !output.status.success() {
        logf!(
            "blkid TYPE {dev}: exit={:?} stdout={out:?} stderr={err_out:?} → treating as no filesystem",
            output.status.code()
        );
        return Ok(String::new());
    }
    Ok(out)
}

pub fn partition_looks_empty(part: &str) -> Result<bool> {
    let dir = tempfile_dir("prepare-nvme-check-")?;
    let _cleanup = DirGuard(dir.clone());

    logf!(
        "emptiness probe: mount -t ext4 -o ro {part} {}",
        dir.display()
    );
    if let Err(e) = run_cmd(
        MOUNT_BIN,
        &["-t", "ext4", "-o", "ro", part, dir.to_str().unwrap()],
    ) {
        logf!("emptiness probe: ext4 mount failed ({e}) — trying mount -o ro without -t");
        if let Err(e2) = run_cmd(MOUNT_BIN, &["-o", "ro", part, dir.to_str().unwrap()]) {
            logf!("emptiness probe: cannot mount {part} ({e2}) — treating as empty/unformatted");
            return Ok(true);
        }
    }

    let empty = dir_empty_except_lost_found(&dir)?;
    let names: Vec<_> = fs::read_dir(&dir)?
        .filter_map(|e| e.ok().map(|e| e.file_name().to_string_lossy().into_owned()))
        .collect();
    logf!("emptiness probe: entries={names:?} empty={empty}");

    if let Err(e) = run_cmd(UMOUNT_BIN, &[dir.to_str().unwrap()]) {
        logf!("warning: umount probe {}: {e}", dir.display());
    }
    Ok(empty)
}

pub fn dir_empty_except_lost_found(dir: &Path) -> Result<bool> {
    for ent in fs::read_dir(dir)? {
        let name = ent?.file_name();
        let name = name.to_string_lossy();
        if name == "." || name == ".." || name == "lost+found" {
            continue;
        }
        return Ok(false);
    }
    Ok(true)
}

pub fn is_mounted(dir: &str) -> bool {
    !mount_source(dir).is_empty()
}

pub fn mount_source(dir: &str) -> String {
    let Ok(content) = fs::read_to_string("/proc/mounts") else {
        return String::new();
    };
    for line in content.lines() {
        let mut fields = line.split_whitespace();
        let Some(dev) = fields.next() else { continue };
        let Some(mp) = fields.next() else { continue };
        if mp == dir {
            return dev.to_string();
        }
    }
    String::new()
}

pub fn describe_mount(dir: &str) -> String {
    let Ok(content) = fs::read_to_string("/proc/mounts") else {
        return "(no /proc/mounts)".into();
    };
    for line in content.lines() {
        let fields: Vec<_> = line.split_whitespace().collect();
        if fields.len() >= 3 && fields[1] == dir {
            let opts = fields.get(3).copied().unwrap_or("");
            return format!("device={} fstype={} opts={opts}", fields[0], fields[2]);
        }
    }
    "(not mounted)".into()
}

pub fn same_device(a: &str, b: &str) -> bool {
    if a.is_empty() || b.is_empty() {
        return false;
    }
    let ra = fs::canonicalize(a).unwrap_or_else(|_| PathBuf::from(a));
    let rb = fs::canonicalize(b).unwrap_or_else(|_| PathBuf::from(b));
    ra == rb
}

pub fn run_cmd(bin: &str, args: &[&str]) -> Result<()> {
    let line = if args.is_empty() {
        bin.to_string()
    } else {
        format!("{bin} {}", args.join(" "))
    };
    logf!("+ {line}");
    let output = Command::new(bin)
        .args(args)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .output()?;
    let out = {
        let mut s = String::new();
        s.push_str(&String::from_utf8_lossy(&output.stdout));
        if !output.stderr.is_empty() {
            if !s.is_empty() && !s.ends_with('\n') {
                s.push('\n');
            }
            s.push_str(&String::from_utf8_lossy(&output.stderr));
        }
        s.trim().to_string()
    };
    if !out.is_empty() {
        for l in out.lines() {
            logf!("  | {l}");
        }
    }
    if !output.status.success() {
        return Err(Error::Msg(format!(
            "{line}: exit {:?}{}",
            output.status.code(),
            format_cmd_out(&out)
        )));
    }
    Ok(())
}

pub fn format_cmd_out(out: &str) -> String {
    if out.is_empty() {
        return String::new();
    }
    let lines: Vec<_> = out.lines().collect();
    let slice = if lines.len() > 8 {
        &lines[lines.len() - 8..]
    } else {
        &lines[..]
    };
    format!("\n--- command output ---\n{}", slice.join("\n"))
}

pub fn log_tools() {
    // fdisk no longer required — GPT via gptman.
    for p in [
        MKFS_EXT4_BIN,
        MKE2FS_BIN,
        BLKID_BIN,
        PARTPROBE_BIN,
        MOUNT_BIN,
        UMOUNT_BIN,
        CP_BIN,
    ] {
        match fs::metadata(p) {
            Ok(st) => logf!("tool {p}: ok mode={:?}", st.permissions()),
            Err(e) => logf!("tool {p}: MISSING ({e})"),
        }
    }
}

pub fn log_disk_info(disk: &str) {
    let output = Command::new(BLKID_BIN).arg(disk).output();
    match output {
        Ok(o) => {
            let out = String::from_utf8_lossy(&o.stdout).trim().to_string();
            if out.is_empty() {
                logf!("blkid {disk}: (no tokens)");
            } else {
                logf!("blkid {disk}: {out}");
            }
        }
        Err(e) => logf!("blkid {disk}: failed to run ({e})"),
    }
    let base = Path::new(disk)
        .file_name()
        .and_then(|s| s.to_str())
        .unwrap_or("");
    if let Ok(b) = fs::read_to_string(format!("/sys/block/{base}/size")) {
        logf!("sysfs {base} size_sectors={}", b.trim());
    }
    // Log existing GPT if present.
    if let Ok(mut f) = File::open(disk) {
        match GPT::find_from(&mut f) {
            Ok(gpt) => {
                let used: Vec<_> = gpt
                    .iter()
                    .filter(|(_, p)| p.is_used())
                    .map(|(i, p)| {
                        format!(
                            "#{i} LBA {}..{} name={:?}",
                            p.starting_lba, p.ending_lba, p.partition_name
                        )
                    })
                    .collect();
                logf!(
                    "gptman find_from: sector_size={} used_partitions={used:?}",
                    gpt.sector_size
                );
            }
            Err(e) => logf!("gptman find_from: no GPT ({e})"),
        }
    }
}

pub fn log_mounts_relevant() {
    let Ok(content) = fs::read_to_string("/proc/mounts") else {
        logf!("cannot read /proc/mounts");
        return;
    };
    logf!("/proc/mounts (data/nvme/mmc relevant):");
    let mut n = 0;
    for line in content.lines() {
        if line.contains("/data")
            || line.contains("nvme")
            || line.contains("mmcblk")
            || line.contains(TMP_MOUNT)
            || line.contains("/etc/shadow")
        {
            logf!("  mounts| {line}");
            n += 1;
        }
    }
    if n == 0 {
        logf!("  mounts| (none matched)");
    }
}

pub fn log_busy_mount(dir: &str) {
    logf!("busy check for {dir}:");
    if let Ok(content) = fs::read_to_string("/proc/mounts") {
        for line in content.lines() {
            let fields: Vec<_> = line.split_whitespace().collect();
            if fields.len() < 2 {
                continue;
            }
            let mp = fields[1];
            if mp == dir || mp.starts_with(&format!("{dir}/")) {
                logf!("  nested/self mount: {line}");
            }
        }
    }
    if Path::new(FUSER_BIN).exists() {
        let output = Command::new(FUSER_BIN).args(["-vm", dir]).output();
        match output {
            Ok(o) => {
                let mut s = String::from_utf8_lossy(&o.stdout).into_owned();
                s.push_str(&String::from_utf8_lossy(&o.stderr));
                let s = s.trim();
                if s.is_empty() {
                    logf!("  fuser: (no output)");
                } else {
                    for l in s.lines() {
                        logf!("  fuser| {l}");
                    }
                }
            }
            Err(e) => logf!("  fuser: run failed ({e})"),
        }
    } else {
        logf!("  fuser: not available at {FUSER_BIN}");
    }
}

pub fn random_guid() -> [u8; 16] {
    let mut g = [0u8; 16];
    if let Err(e) = getrandom::fill(&mut g) {
        // Extremely unlikely; fall back to time-based noise.
        logf!("warning: getrandom failed ({e}) — using weak GUID");
        let t = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0);
        for (i, b) in g.iter_mut().enumerate() {
            *b = ((t >> (i * 8)) as u8) ^ (i as u8).wrapping_mul(17);
        }
    }
    g
}

pub fn guid_hex(g: &[u8; 16]) -> String {
    g.iter()
        .map(|b| format!("{b:02x}"))
        .collect::<Vec<_>>()
        .join("")
}

pub fn tempfile_dir(prefix: &str) -> Result<PathBuf> {
    let mut path = std::env::temp_dir();
    let unique = format!(
        "{prefix}{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0)
    );
    path.push(unique);
    fs::create_dir(&path)?;
    Ok(path)
}

pub struct DirGuard(pub PathBuf);
impl Drop for DirGuard {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use gptman::{GPTPartitionEntry, GPT};
    use std::io;

    #[test]
    fn dir_empty_except_lost_found_works() {
        let dir = tempfile_dir("pn-test-").unwrap();
        assert!(dir_empty_except_lost_found(&dir).unwrap());
        fs::create_dir(dir.join("lost+found")).unwrap();
        assert!(dir_empty_except_lost_found(&dir).unwrap());
        fs::write(dir.join("keep"), b"x").unwrap();
        assert!(!dir_empty_except_lost_found(&dir).unwrap());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn partition_index_and_sort() {
        let mut parts = vec![
            "/dev/nvme0n1p10".into(),
            "/dev/nvme0n1p2".into(),
            "/dev/nvme0n1p1".into(),
        ];
        sort_partitions(&mut parts, "nvme0n1");
        assert_eq!(
            parts,
            vec![
                "/dev/nvme0n1p1".to_string(),
                "/dev/nvme0n1p2".to_string(),
                "/dev/nvme0n1p10".to_string(),
            ]
        );
    }

    #[test]
    fn same_device_basic() {
        assert!(same_device("/dev/nvme0n1p1", "/dev/nvme0n1p1"));
        assert!(!same_device("/dev/nvme0n1p1", "/dev/mmcblk0p3"));
        assert!(!same_device("", "/dev/nvme0n1p1"));
    }

    #[test]
    fn gptman_creates_table_in_memory() {
        let ss = 512u64;
        let data = vec![0u8; 100 * ss as usize];
        let mut cur = io::Cursor::new(data);
        let mut gpt = GPT::new_from(&mut cur, ss, [0x11; 16]).unwrap();
        gpt[1] = GPTPartitionEntry {
            partition_type_guid: LINUX_FS_GUID,
            unique_partition_guid: [0x22; 16],
            starting_lba: gpt.header.first_usable_lba,
            ending_lba: gpt.header.last_usable_lba,
            attribute_bits: 0,
            partition_name: "bigfred-data".into(),
        };
        gpt.write_into(&mut cur).unwrap();
        let mut cur2 = io::Cursor::new(cur.into_inner());
        let read = GPT::find_from(&mut cur2).unwrap();
        assert!(read[1].is_used());
        assert_eq!(read[1].partition_type_guid, LINUX_FS_GUID);
    }
}
