// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Erases everything the embedded Core has stored.

use std::io;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use super::{Config, backup};

/// The suffix of a directory that was retired and waits for removal.
const ERASED: &str = ".erased";

/// Erases the Core data and its backups, and leaves an empty data directory. Call it
/// only while no Core runs. Each directory is first renamed, which is atomic, so a
/// failure never leaves data that is half erased under the name the Core opens.
pub fn run(cfg: &Config) -> io::Result<()> {
    for dir in targets(cfg) {
        retire(&dir)?;
    }
    std::fs::create_dir_all(&cfg.data_dir)?;
    sweep(cfg)
}

/// Removes every retired directory that an earlier [`run`] could not remove.
pub fn sweep(cfg: &Config) -> io::Result<()> {
    for dir in targets(cfg) {
        let Some(parent) = dir.parent() else { continue };
        let entries = match std::fs::read_dir(parent) {
            Ok(entries) => entries,
            Err(err) if err.kind() == io::ErrorKind::NotFound => continue,
            Err(err) => return Err(err),
        };
        for entry in entries {
            let path = entry?.path();
            if path.to_string_lossy().ends_with(ERASED) {
                std::fs::remove_dir_all(path)?;
            }
        }
    }
    Ok(())
}

fn targets(cfg: &Config) -> [PathBuf; 2] {
    [cfg.data_dir.clone(), backup::dir(cfg)]
}

fn retire(dir: &Path) -> io::Result<()> {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(io::Error::other)?
        .as_nanos();
    let mut retired = dir.as_os_str().to_owned();
    retired.push(format!(".{nanos}{ERASED}"));
    match std::fs::rename(dir, retired) {
        Err(err) if err.kind() != io::ErrorKind::NotFound => Err(err),
        _ => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn cfg(root: &Path) -> Config {
        Config::new(
            root.join("program"),
            "0.58.0".to_string(),
            root.join("work"),
            root.join("work").join("core"),
            root.join("logs"),
        )
    }

    #[test]
    fn erases_the_data_and_the_backups_and_keeps_the_rest() {
        let root = tempfile::tempdir().unwrap();
        let cfg = cfg(root.path());
        std::fs::create_dir_all(cfg.data_dir.join("kv")).unwrap();
        std::fs::write(cfg.data_dir.join("kv").join("MANIFEST"), "data").unwrap();
        std::fs::create_dir_all(backup::dir(&cfg).join("0000000001-0.57.0")).unwrap();
        std::fs::write(cfg.work_dir.join("session.json"), "{}").unwrap();

        run(&cfg).unwrap();

        let mut left: Vec<String> = std::fs::read_dir(&cfg.work_dir)
            .unwrap()
            .map(|e| e.unwrap().file_name().to_string_lossy().into_owned())
            .collect();
        left.sort();
        assert_eq!(left, ["core", "session.json"]);
        assert_eq!(std::fs::read_dir(&cfg.data_dir).unwrap().count(), 0);
    }

    #[test]
    fn succeeds_when_there_is_nothing_to_erase() {
        let root = tempfile::tempdir().unwrap();
        let cfg = cfg(root.path());
        run(&cfg).unwrap();
        assert!(cfg.data_dir.is_dir());
    }

    #[test]
    fn sweeps_a_retired_directory_that_was_left_behind() {
        let root = tempfile::tempdir().unwrap();
        let cfg = cfg(root.path());
        let retired = cfg.work_dir.join("core.1.erased");
        std::fs::create_dir_all(retired.join("kv")).unwrap();
        sweep(&cfg).unwrap();
        assert!(!retired.exists());
    }
}
