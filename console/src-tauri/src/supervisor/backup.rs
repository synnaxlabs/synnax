// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Backs up the metadata store before a new version of the Core migrates it.

use std::io;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use super::Config;

/// The directory of the metadata store inside the Core data directory. Telemetry is
/// left out: it is large, and the metadata is what a user cannot record again.
const STORE: &str = "kv";

/// The number of backups to keep.
const KEEP: usize = 3;

/// The suffix of a backup whose copy has not finished.
const PARTIAL: &str = ".partial";

/// Returns the directory that holds the backups.
pub fn dir(cfg: &Config) -> PathBuf {
    cfg.work_dir.join("backups")
}

/// Copies the metadata store aside when the app version differs from the version that
/// last ran on the data, then records the app version. It does nothing on any later
/// call for the same version. Call it only while no Core runs.
pub fn run(cfg: &Config) -> io::Result<()> {
    let marker = cfg.work_dir.join("core.version");
    let last = read_version(&marker)?;
    if last.as_deref() == Some(cfg.version.as_str()) {
        return Ok(());
    }
    let store = cfg.data_dir.join(STORE);
    if store.is_dir() {
        let dir = dir(cfg);
        std::fs::create_dir_all(&dir)?;
        let seconds = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(io::Error::other)?
            .as_secs();
        // The time leads the name, so the names sort from oldest to newest.
        let name = format!("{seconds:010}-{}", last.as_deref().unwrap_or("unknown"));
        let partial = dir.join(format!("{name}{PARTIAL}"));
        copy_dir(&store, &partial.join(STORE))?;
        std::fs::rename(&partial, dir.join(name))?;
        prune(&dir)?;
    }
    std::fs::write(marker, &cfg.version)
}

fn read_version(marker: &Path) -> io::Result<Option<String>> {
    match std::fs::read_to_string(marker) {
        Ok(version) => Ok(Some(version.trim().to_string())),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(err),
    }
}

fn copy_dir(from: &Path, to: &Path) -> io::Result<()> {
    std::fs::create_dir_all(to)?;
    for entry in std::fs::read_dir(from)? {
        let entry = entry?;
        let dest = to.join(entry.file_name());
        if entry.file_type()?.is_dir() {
            copy_dir(&entry.path(), &dest)?;
        } else {
            std::fs::copy(entry.path(), dest)?;
        }
    }
    Ok(())
}

/// Removes unfinished backups and all but the newest [`KEEP`] finished ones.
fn prune(dir: &Path) -> io::Result<()> {
    let mut finished: Vec<PathBuf> = Vec::new();
    for entry in std::fs::read_dir(dir)? {
        let path = entry?.path();
        if path.to_string_lossy().ends_with(PARTIAL) {
            std::fs::remove_dir_all(&path)?;
        } else if path.is_dir() {
            finished.push(path);
        }
    }
    finished.sort();
    let excess = finished.len().saturating_sub(KEEP);
    for path in &finished[..excess] {
        std::fs::remove_dir_all(path)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Fixture {
        dir: tempfile::TempDir,
    }

    impl Fixture {
        fn new() -> Self {
            Self {
                dir: tempfile::tempdir().unwrap(),
            }
        }

        fn cfg(&self, version: &str) -> Config {
            Config::new(
                self.dir.path().join("program"),
                version.to_string(),
                self.dir.path().join("work"),
                self.dir.path().join("data"),
                self.dir.path().join("logs"),
            )
        }

        fn write_store(&self, contents: &str) {
            let nested = self.dir.path().join("data").join(STORE).join("nested");
            std::fs::create_dir_all(&nested).unwrap();
            std::fs::write(nested.join("MANIFEST"), contents).unwrap();
        }

        fn backups(&self) -> Vec<PathBuf> {
            let mut backups: Vec<PathBuf> =
                std::fs::read_dir(self.dir.path().join("work").join("backups"))
                    .map(|entries| entries.map(|e| e.unwrap().path()).collect())
                    .unwrap_or_default();
            backups.sort();
            backups
        }
    }

    fn open(f: &Fixture, version: &str) {
        let cfg = f.cfg(version);
        std::fs::create_dir_all(&cfg.work_dir).unwrap();
        run(&cfg).unwrap();
    }

    #[test]
    fn records_the_version_of_a_first_launch_without_a_backup() {
        let f = Fixture::new();
        open(&f, "0.58.0");
        assert!(f.backups().is_empty());
        let marker = f.dir.path().join("work").join("core.version");
        assert_eq!(std::fs::read_to_string(marker).unwrap(), "0.58.0");
    }

    #[test]
    fn copies_the_store_when_the_version_changes() {
        let f = Fixture::new();
        open(&f, "0.58.0");
        f.write_store("before the update");
        open(&f, "0.59.0");
        let backups = f.backups();
        assert_eq!(backups.len(), 1);
        let name = backups[0].file_name().unwrap().to_string_lossy();
        assert!(name.ends_with("-0.58.0"), "{name}");
        let copied = backups[0].join(STORE).join("nested").join("MANIFEST");
        assert_eq!(
            std::fs::read_to_string(copied).unwrap(),
            "before the update"
        );
    }

    #[test]
    fn copies_nothing_on_a_later_launch_of_the_same_version() {
        let f = Fixture::new();
        f.write_store("data");
        open(&f, "0.58.0");
        open(&f, "0.58.0");
        assert_eq!(f.backups().len(), 1);
    }

    #[test]
    fn labels_the_backup_of_data_without_a_recorded_version() {
        let f = Fixture::new();
        f.write_store("data");
        open(&f, "0.58.0");
        let name = f.backups()[0]
            .file_name()
            .unwrap()
            .to_string_lossy()
            .into_owned();
        assert!(name.ends_with("-unknown"), "{name}");
    }

    #[test]
    fn keeps_the_newest_backups_and_removes_unfinished_ones() {
        let f = Fixture::new();
        let dir = f.dir.path().join("work").join("backups");
        for name in [
            "0000000001-a",
            "0000000002-b",
            "0000000003-c",
            "0000000004-d.partial",
        ] {
            std::fs::create_dir_all(dir.join(name)).unwrap();
        }
        f.write_store("data");
        open(&f, "0.58.0");
        let names: Vec<String> = f
            .backups()
            .iter()
            .map(|p| p.file_name().unwrap().to_string_lossy().into_owned())
            .collect();
        assert_eq!(names.len(), KEEP);
        assert_eq!(names[0], "0000000002-b");
        assert_eq!(names[1], "0000000003-c");
        assert!(names[2].ends_with("-unknown"), "{}", names[2]);
    }
}
