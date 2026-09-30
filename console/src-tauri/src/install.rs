// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Identifies the install across launches, and reports what this launch knows about
//! the one before it.

use std::io;
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, Runtime, State};

/// The file that holds the record, in the app local data directory. A reset erases the
/// Core data directory and the backups, so the install survives one.
const FILE: &str = "install.json";

const HOUR: f64 = 3600.0;

/// What the file holds between two launches.
#[derive(Deserialize, Serialize)]
struct Stored {
    id: String,
    /// The time of the launch that wrote the file, in seconds since the Unix epoch.
    launched_at: u64,
    /// The bytes a reset erased since the last launch, which the next launch reports.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    erased_bytes: Option<u64>,
}

/// What this install is, and what this launch knows about the one before it.
#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Info {
    /// Identifies the install across launches.
    pub id: String,
    /// True when this launch minted the identifier.
    pub first_launch: bool,
    /// The hours between this launch and the one before it. None on a first launch.
    pub hours_since_last_launch: Option<f64>,
    /// The bytes a reset erased before this launch. None when no reset ran.
    pub erased_bytes: Option<u64>,
    pub os: &'static str,
    pub arch: &'static str,
}

/// The record of this launch, or why it could not be read or written.
pub struct Record(pub Result<Info, String>);

/// Reads the record of this install, mints one on a first launch, and records this
/// launch. Manages the result as app state, so every window reports the same launch. A
/// launch without a record still runs; only `install_info` fails.
pub fn init<R: Runtime>(app: &AppHandle<R>) {
    let dir = app
        .path()
        .app_local_data_dir()
        .map_err(|err| err.to_string());
    app.manage(Record(dir.and_then(|dir| load(&dir))));
}

#[tauri::command]
pub fn install_info(record: State<'_, Record>) -> Result<Info, String> {
    record.0.clone()
}

fn load(dir: &Path) -> Result<Info, String> {
    now()
        .and_then(|now| open(dir, now))
        .map_err(|err| format!("failed to record the install: {err}"))
}

/// Records that a reset is about to erase `bytes`, for the next launch to report.
pub fn record_reset(dir: &Path, bytes: u64) -> io::Result<()> {
    let mut stored = read(dir).ok_or_else(|| io::Error::other("no install record"))?;
    stored.erased_bytes = Some(bytes);
    write(dir, &stored)
}

fn read(dir: &Path) -> Option<Stored> {
    let bytes = std::fs::read(dir.join(FILE)).ok()?;
    serde_json::from_slice(&bytes).ok()
}

/// Returns the record in `dir`, then writes it back with `now` as the launch time. A
/// file that does not parse is replaced, which reads as a new install.
fn open(dir: &Path, now: u64) -> io::Result<Info> {
    std::fs::create_dir_all(dir)?;
    let stored = read(dir);
    let info = Info {
        id: match &stored {
            Some(stored) => stored.id.clone(),
            None => id()?,
        },
        first_launch: stored.is_none(),
        hours_since_last_launch: stored
            .as_ref()
            .map(|stored| now.saturating_sub(stored.launched_at) as f64 / HOUR),
        erased_bytes: stored.and_then(|stored| stored.erased_bytes),
        os: std::env::consts::OS,
        arch: std::env::consts::ARCH,
    };
    write(
        dir,
        &Stored {
            id: info.id.clone(),
            launched_at: now,
            erased_bytes: None,
        },
    )?;
    Ok(info)
}

/// Writes `stored` to `dir` through a rename, which is atomic, so a crash never leaves
/// a half written record that would read as a new install.
fn write(dir: &Path, stored: &Stored) -> io::Result<()> {
    let temp = dir.join(format!("{FILE}.tmp"));
    std::fs::write(&temp, serde_json::to_vec(stored).map_err(io::Error::other)?)?;
    std::fs::rename(temp, dir.join(FILE))
}

/// Returns a new identifier: 16 bytes, hex encoded.
fn id() -> io::Result<String> {
    let mut bytes = [0u8; 16];
    getrandom::fill(&mut bytes).map_err(|err| io::Error::other(err.to_string()))?;
    Ok(bytes.iter().map(|b| format!("{b:02x}")).collect())
}

fn now() -> io::Result<u64> {
    Ok(SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(io::Error::other)?
        .as_secs())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mints_an_identifier_on_a_first_launch() {
        let root = tempfile::tempdir().unwrap();
        let info = open(root.path(), 1_000).unwrap();
        assert_eq!(info.id.len(), 32);
        assert!(info.first_launch);
        assert_eq!(info.hours_since_last_launch, None);
    }

    #[test]
    fn keeps_the_identifier_across_launches() {
        let root = tempfile::tempdir().unwrap();
        let first = open(root.path(), 1_000).unwrap();
        let second = open(root.path(), 1_000 + 7_200).unwrap();
        assert_eq!(second.id, first.id);
        assert!(!second.first_launch);
        assert_eq!(second.hours_since_last_launch, Some(2.0));
    }

    #[test]
    fn replaces_a_record_that_does_not_parse() {
        let root = tempfile::tempdir().unwrap();
        std::fs::write(root.path().join(FILE), "not json").unwrap();
        let info = open(root.path(), 1_000).unwrap();
        assert!(info.first_launch);
        assert_eq!(open(root.path(), 2_000).unwrap().id, info.id);
    }

    #[test]
    fn reports_a_reset_on_the_next_launch_only() {
        let root = tempfile::tempdir().unwrap();
        let first = open(root.path(), 1_000).unwrap();
        assert_eq!(first.erased_bytes, None);
        record_reset(root.path(), 4_096).unwrap();
        let second = open(root.path(), 2_000).unwrap();
        assert_eq!(second.id, first.id);
        assert_eq!(second.erased_bytes, Some(4_096));
        assert_eq!(open(root.path(), 3_000).unwrap().erased_bytes, None);
    }

    #[test]
    fn reports_why_a_record_cannot_be_written() {
        let root = tempfile::tempdir().unwrap();
        let file = root.path().join("file");
        std::fs::write(&file, "").unwrap();
        let err = load(&file.join("install")).err().unwrap();
        assert!(err.starts_with("failed to record the install: "), "{err}");
    }

    #[test]
    fn reports_no_time_since_a_launch_of_the_future() {
        let root = tempfile::tempdir().unwrap();
        open(root.path(), 9_000).unwrap();
        // A clock that moves backwards, which a timezone change or an NTP correction
        // can do, must not report a negative age.
        assert_eq!(
            open(root.path(), 1_000).unwrap().hours_since_last_launch,
            Some(0.0)
        );
    }
}
