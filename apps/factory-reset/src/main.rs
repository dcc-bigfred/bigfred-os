//! factory-reset — destructive NVMe wipe + ext4 reformat for BigFred OS.
//!
//! Operator tool: destroys all data on the local NVMe disk, writes a fresh
//! GPT + single ext4 partition (`LABEL=bigfred-data`), and does **not** write
//! the `.bigfred-nvme` migration marker so the next boot falls back to microSD
//! `/data`. Re-run `prepare-nvme` to migrate again.

use std::env;
use std::fs;
use std::io::{self, BufRead, Write};
use std::path::Path;
use std::process;

use prepare_nvme::{
    find_nvme_disk, is_mounted, log_disk_info, log_mounts_relevant, mount_source, same_device,
    unmount_disk_mounts, wipe_format_data_partition, BIGFRED_MARKER, DATA_MOUNT,
};

const LSB_RELEASE: &str = "/etc/lsb-release";
const EXPECTED_DISTRIB_ID: &str = "bigfred-os";

#[derive(Debug, Default)]
struct Args {
    yes: bool,
    dry_run: bool,
    help: bool,
}

fn main() {
    let args = match parse_args(env::args().skip(1)) {
        Ok(a) => a,
        Err(msg) => {
            eprintln!("factory-reset: {msg}");
            usage();
            process::exit(2);
        }
    };
    if args.help {
        usage();
        return;
    }

    if let Err(e) = require_bigfred_os(LSB_RELEASE, BIGFRED_MARKER) {
        eprintln!("factory-reset: ERROR: {e}");
        process::exit(1);
    }

    if let Err(e) = run(args) {
        eprintln!("factory-reset: ERROR: {e}");
        log_mounts_relevant();
        process::exit(1);
    }
}

fn run(args: Args) -> Result<(), String> {
    let disk = find_nvme_disk().map_err(|e| e.to_string())?;
    let Some(disk) = disk else {
        eprintln!("factory-reset: no NVMe disk under /sys/block/nvme* — nothing to do");
        return Ok(());
    };

    eprintln!("factory-reset: target disk {disk}");
    log_disk_info(&disk);
    log_mounts_relevant();

    if args.dry_run {
        eprintln!(
            "factory-reset: dry-run — would DESTROY all data on {disk}, \
             create a new GPT partition, format ext4 (LABEL=bigfred-data), \
             and leave no .bigfred-nvme marker"
        );
        if is_mounted(DATA_MOUNT) {
            let src = mount_source(DATA_MOUNT);
            eprintln!(
                "factory-reset: dry-run — current {DATA_MOUNT} is on {src} ({})",
                if src_is_under_disk(&src, &disk) {
                    "would unmount first"
                } else {
                    "leave mounted (not on NVMe)"
                }
            );
        }
        return Ok(());
    }

    if !args.yes {
        confirm_wipe(&disk)?;
    }

    // Tear down NVMe mounts (including /data when it lives on NVMe).
    if is_mounted(DATA_MOUNT) {
        let src = mount_source(DATA_MOUNT);
        if src_is_under_disk(&src, &disk) {
            eprintln!("factory-reset: {DATA_MOUNT} is on {src} — unmounting NVMe mounts");
        } else {
            eprintln!(
                "factory-reset: {DATA_MOUNT} is on {src} (not NVMe) — leaving it mounted"
            );
        }
    }
    unmount_disk_mounts(&disk).map_err(|e| e.to_string())?;

    let part = wipe_format_data_partition(&disk).map_err(|e| e.to_string())?;
    eprintln!(
        "factory-reset: {disk} wiped and reformatted ({part}, ext4, LABEL=bigfred-data)."
    );
    eprintln!(
        "factory-reset: reboot to return to microSD {DATA_MOUNT}. \
         Run prepare-nvme to re-migrate."
    );
    Ok(())
}

fn src_is_under_disk(src: &str, disk: &str) -> bool {
    if src.is_empty() {
        return false;
    }
    same_device(src, disk) || src.starts_with(&format!("{disk}p"))
}

fn confirm_wipe(disk: &str) -> Result<(), String> {
    let mut stderr = io::stderr().lock();
    writeln!(
        stderr,
        "WARNING: this will DESTROY ALL DATA on {disk}.\n\
         Current {DATA_MOUNT} will fall back to microSD on next boot.\n\
         Type \"yes\" to continue:"
    )
    .map_err(|e| e.to_string())?;
    stderr.flush().map_err(|e| e.to_string())?;

    let mut line = String::new();
    io::stdin()
        .lock()
        .read_line(&mut line)
        .map_err(|e| format!("read confirmation: {e}"))?;
    if line.trim() != "yes" {
        return Err("aborted (confirmation was not exactly \"yes\")".into());
    }
    Ok(())
}

/// Refuse to run unless this looks like a BigFred OS image.
fn require_bigfred_os(lsb_path: &str, marker_dir: &str) -> Result<(), String> {
    let body = fs::read_to_string(lsb_path).map_err(|_| {
        format!("{lsb_path} not found — refusing to run outside BigFred OS")
    })?;
    if !distrib_id_is(&body, EXPECTED_DISTRIB_ID) {
        return Err(format!(
            "{lsb_path} DISTRIB_ID is not {EXPECTED_DISTRIB_ID} — refusing to run"
        ));
    }
    if !Path::new(marker_dir).is_dir() {
        return Err(format!(
            "{marker_dir} not found — refusing to run outside BigFred OS"
        ));
    }
    Ok(())
}

fn distrib_id_is(lsb_body: &str, want: &str) -> bool {
    for line in lsb_body.lines() {
        let line = line.trim();
        let Some(rest) = line.strip_prefix("DISTRIB_ID=") else {
            continue;
        };
        let val = rest.trim().trim_matches('"');
        return val == want;
    }
    false
}

fn parse_args<I, S>(args: I) -> Result<Args, String>
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    let mut out = Args::default();
    for raw in args {
        match raw.as_ref() {
            "-h" | "--help" | "help" => out.help = true,
            "--yes" | "-y" => out.yes = true,
            "--dry-run" => out.dry_run = true,
            other => return Err(format!("unknown argument {other:?}")),
        }
    }
    Ok(out)
}

fn usage() {
    let prog = env::args().next().unwrap_or_else(|| "factory-reset".into());
    eprintln!("usage: {prog} [--yes] [--dry-run]");
    eprintln!("  Destructive wipe of the local NVMe disk (GPT + ext4 LABEL=bigfred-data).");
    eprintln!("  Does not write .bigfred-nvme — next boot uses microSD /data.");
    eprintln!("  --yes       skip interactive confirmation");
    eprintln!("  --dry-run   print what would happen, do not modify disks");
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    #[test]
    fn parse_args_flags() {
        let a = parse_args(["--yes", "--dry-run"]).unwrap();
        assert!(a.yes && a.dry_run && !a.help);
        let h = parse_args(["--help"]).unwrap();
        assert!(h.help);
        assert!(parse_args(["--nope"]).is_err());
    }

    #[test]
    fn distrib_id_parsing() {
        let body = "DISTRIB_ID=bigfred-os\nDISTRIB_RELEASE=rolling\n";
        assert!(distrib_id_is(body, "bigfred-os"));
        assert!(!distrib_id_is(body, "buildroot"));
        assert!(distrib_id_is("DISTRIB_ID=\"bigfred-os\"\n", "bigfred-os"));
        assert!(!distrib_id_is("DISTRIB_RELEASE=rolling\n", "bigfred-os"));
    }

    #[test]
    fn require_bigfred_os_ok() {
        let dir = tempfile_dir();
        let lsb = dir.join("lsb-release");
        fs::write(&lsb, "DISTRIB_ID=bigfred-os\nDISTRIB_RELEASE=rolling\n").unwrap();
        let marker = dir.join("marker");
        fs::create_dir(&marker).unwrap();
        require_bigfred_os(lsb.to_str().unwrap(), marker.to_str().unwrap()).unwrap();
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn require_bigfred_os_rejects_wrong_id() {
        let dir = tempfile_dir();
        let lsb = dir.join("lsb-release");
        fs::write(&lsb, "DISTRIB_ID=buildroot\n").unwrap();
        let marker = dir.join("marker");
        fs::create_dir(&marker).unwrap();
        assert!(require_bigfred_os(lsb.to_str().unwrap(), marker.to_str().unwrap()).is_err());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn src_is_under_disk_paths() {
        assert!(src_is_under_disk("/dev/nvme0n1p1", "/dev/nvme0n1"));
        assert!(!src_is_under_disk("/dev/mmcblk0p3", "/dev/nvme0n1"));
        assert!(!src_is_under_disk("", "/dev/nvme0n1"));
    }

    fn tempfile_dir() -> PathBuf {
        let mut path = env::temp_dir();
        path.push(format!(
            "factory-reset-test-{}-{}",
            process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_nanos())
                .unwrap_or(0)
        ));
        fs::create_dir(&path).unwrap();
        path
    }
}
