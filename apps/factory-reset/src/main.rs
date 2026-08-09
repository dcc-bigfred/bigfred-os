//! factory-reset — wipe hub `/data` contents and reboot on BigFred OS.
//!
//! `/data` itself stays mounted (busy). Nested mounts under `/data/…`
//! (notably tmpfs `/data/logs`) are unmounted first, then every entry under
//! `/data` is removed. Finally `shutdown -r now` so early-boot reseeds layout.

use std::env;
use std::fs;
use std::io::{self, BufRead, Write};
use std::path::Path;
use std::process::{self, Command};

const DATA_MOUNT: &str = "/data";
const LSB_RELEASE: &str = "/etc/lsb-release";
const BIGFRED_MARKER: &str = "/var/lib/bigfred";
const EXPECTED_DISTRIB_ID: &str = "bigfred-os";
const UMOUNT_BIN: &str = "/bin/umount";
const SHUTDOWN_BIN: &str = "/usr/sbin/shutdown";

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
        process::exit(1);
    }
}

fn run(args: Args) -> Result<(), String> {
    if !Path::new(DATA_MOUNT).is_dir() {
        return Err(format!("{DATA_MOUNT} is missing"));
    }

    let nested = nested_mounts_under(DATA_MOUNT)?;
    eprintln!("factory-reset: nested mounts under {DATA_MOUNT}: {nested:?}");

    if args.dry_run {
        eprintln!(
            "factory-reset: dry-run — would umount nested mounts, \
             delete all entries under {DATA_MOUNT}, then `{SHUTDOWN_BIN} -r now`"
        );
        return Ok(());
    }

    if !args.yes {
        confirm_wipe()?;
    }

    for mp in &nested {
        eprintln!("factory-reset: umount {mp}");
        umount(mp)?;
    }

    wipe_data_contents(DATA_MOUNT)?;
    eprintln!("factory-reset: {DATA_MOUNT} cleared");

    eprintln!("factory-reset: rebooting (`{SHUTDOWN_BIN} -r now`)");
    reboot_now()?;
    Ok(())
}

/// Mountpoints strictly under `root/` (not `root` itself), longest first.
fn nested_mounts_under(root: &str) -> Result<Vec<String>, String> {
    let content = fs::read_to_string("/proc/mounts")
        .map_err(|e| format!("read /proc/mounts: {e}"))?;
    let prefix = format!("{root}/");
    let mut out: Vec<String> = Vec::new();
    for line in content.lines() {
        let mut fields = line.split_whitespace();
        let Some(_dev) = fields.next() else { continue };
        let Some(mp) = fields.next() else { continue };
        if mp.starts_with(&prefix) && !out.iter().any(|m| m == mp) {
            out.push(mp.to_string());
        }
    }
    out.sort_by_key(|m| std::cmp::Reverse(m.len()));
    Ok(out)
}

fn umount(mp: &str) -> Result<(), String> {
    let status = Command::new(UMOUNT_BIN)
        .arg(mp)
        .status()
        .map_err(|e| format!("umount {mp}: {e}"))?;
    if !status.success() {
        return Err(format!("umount {mp}: exit {:?}", status.code()));
    }
    Ok(())
}

fn wipe_data_contents(root: &str) -> Result<(), String> {
    let entries = fs::read_dir(root).map_err(|e| format!("read_dir {root}: {e}"))?;
    for ent in entries {
        let ent = ent.map_err(|e| format!("read_dir {root}: {e}"))?;
        let path = ent.path();
        eprintln!("factory-reset: remove {}", path.display());
        if ent.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            fs::remove_dir_all(&path)
                .map_err(|e| format!("remove_dir_all {}: {e}", path.display()))?;
        } else {
            fs::remove_file(&path).map_err(|e| format!("remove_file {}: {e}", path.display()))?;
        }
    }
    Ok(())
}

fn reboot_now() -> Result<(), String> {
    let status = Command::new(SHUTDOWN_BIN)
        .args(["-r", "now"])
        .status()
        .map_err(|e| format!("{SHUTDOWN_BIN} -r now: {e}"))?;
    if !status.success() {
        return Err(format!(
            "{SHUTDOWN_BIN} -r now: exit {:?}",
            status.code()
        ));
    }
    Ok(())
}

fn confirm_wipe() -> Result<(), String> {
    let mut stderr = io::stderr().lock();
    writeln!(
        stderr,
        "WARNING: this will DESTROY ALL DATA under {DATA_MOUNT} and reboot.\n\
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
    eprintln!("  Unmount nested filesystems under {DATA_MOUNT}, delete its contents, reboot.");
    eprintln!("  --yes       skip interactive confirmation");
    eprintln!("  --dry-run   print what would happen, do not modify or reboot");
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
    fn wipe_data_contents_removes_entries() {
        let dir = tempfile_dir();
        fs::create_dir(dir.join("etc")).unwrap();
        fs::write(dir.join("marker"), b"x").unwrap();
        wipe_data_contents(dir.to_str().unwrap()).unwrap();
        assert!(fs::read_dir(&dir).unwrap().next().is_none());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn nested_mounts_sorts_longest_first() {
        let mut out = vec![
            "/data/a".to_string(),
            "/data/a/b/c".to_string(),
            "/data/logs".to_string(),
        ];
        out.sort_by_key(|m| std::cmp::Reverse(m.len()));
        assert_eq!(out[0], "/data/a/b/c");
        assert!(out[1].len() >= out[2].len());
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
