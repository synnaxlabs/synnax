// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Reads what a person needs to find out why the embedded Core misbehaves: the size of
//! its data, the end of its log, and an archive of all of it for support.

use std::fs::File;
use std::io::{self, Read, Seek, SeekFrom, Write};
use std::path::Path;

use serde::Serialize;
use zip::write::SimpleFileOptions;

use super::{History, Status};

/// The name of the Core log inside the log directory.
const LOG: &str = "core.log";

/// The most bytes [`log_tail`] reads from the end of the log.
const TAIL_BYTES: u64 = 64 * 1024;

/// Returns the total size in bytes of the files under `dir`, or zero when it is absent.
pub fn dir_size(dir: &Path) -> io::Result<u64> {
    let entries = match std::fs::read_dir(dir) {
        Ok(entries) => entries,
        Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(0),
        Err(err) => return Err(err),
    };
    let mut size = 0;
    for entry in entries {
        let entry = entry?;
        let meta = entry.metadata()?;
        size += if meta.is_dir() {
            dir_size(&entry.path())?
        } else {
            meta.len()
        };
    }
    Ok(size)
}

/// Returns the last whole lines of the Core log, or an empty string when it is absent.
pub fn log_tail(log_dir: &Path) -> io::Result<String> {
    let mut file = match File::open(log_dir.join(LOG)) {
        Ok(file) => file,
        Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(String::new()),
        Err(err) => return Err(err),
    };
    let len = file.metadata()?.len();
    let start = len.saturating_sub(TAIL_BYTES);
    file.seek(SeekFrom::Start(start))?;
    let mut bytes = Vec::new();
    file.read_to_end(&mut bytes)?;
    let text = String::from_utf8_lossy(&bytes);
    // A read that starts inside the file most likely starts inside a line.
    let whole = match (start, text.find('\n')) {
        (0, _) | (_, None) => &text[..],
        (_, Some(i)) => &text[i + 1..],
    };
    Ok(whole.to_string())
}

/// The summary at the head of a diagnostics archive.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct Summary<'a> {
    version: &'a str,
    os: &'static str,
    arch: &'static str,
    state: serde_json::Value,
    history: &'a History,
    data_size: u64,
}

/// Writes a zip archive to `dest` with a summary and every file of the log directory.
/// The archive never holds the launch password.
pub fn export(
    dest: &Path,
    version: &str,
    status: &Status,
    history: &History,
    data_dir: &Path,
    log_dir: &Path,
) -> io::Result<()> {
    let mut state = serde_json::to_value(status).map_err(io::Error::other)?;
    if let Some(fields) = state.as_object_mut() {
        fields.remove("connection");
    }
    let summary = Summary {
        version,
        os: std::env::consts::OS,
        arch: std::env::consts::ARCH,
        state,
        history,
        data_size: dir_size(data_dir)?,
    };
    let mut archive = zip::ZipWriter::new(File::create(dest)?);
    let options = SimpleFileOptions::default();
    archive
        .start_file("diagnostics.json", options)
        .map_err(io::Error::other)?;
    archive.write_all(&serde_json::to_vec_pretty(&summary).map_err(io::Error::other)?)?;
    if log_dir.is_dir() {
        for entry in std::fs::read_dir(log_dir)? {
            let entry = entry?;
            if !entry.file_type()?.is_file() {
                continue;
            }
            let name = format!("logs/{}", entry.file_name().to_string_lossy());
            archive
                .start_file(name, options)
                .map_err(io::Error::other)?;
            io::copy(&mut File::open(entry.path())?, &mut archive)?;
        }
    }
    archive.finish().map_err(io::Error::other)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::supervisor::Connection;

    #[test]
    fn sums_the_files_of_a_tree_and_counts_an_absent_one_as_empty() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(dir.path().join("a/b")).unwrap();
        std::fs::write(dir.path().join("a/one"), [0; 10]).unwrap();
        std::fs::write(dir.path().join("a/b/two"), [0; 32]).unwrap();
        assert_eq!(dir_size(dir.path()).unwrap(), 42);
        assert_eq!(dir_size(&dir.path().join("absent")).unwrap(), 0);
    }

    #[test]
    fn returns_a_short_log_whole_and_an_absent_one_as_empty() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(log_tail(dir.path()).unwrap(), "");
        std::fs::write(dir.path().join(LOG), "first\nsecond\n").unwrap();
        assert_eq!(log_tail(dir.path()).unwrap(), "first\nsecond\n");
    }

    #[test]
    fn returns_only_whole_lines_from_the_end_of_a_long_log() {
        let dir = tempfile::tempdir().unwrap();
        let line = "x".repeat(99) + "\n";
        std::fs::write(dir.path().join(LOG), line.repeat(1000) + "last\n").unwrap();
        let tail = log_tail(dir.path()).unwrap();
        assert!(tail.len() as u64 <= TAIL_BYTES);
        assert!(tail.ends_with("last\n"));
        assert!(tail.lines().all(|l| l == "last" || l.len() == 99));
    }

    #[test]
    fn archives_the_logs_and_a_summary_without_the_password() {
        let dir = tempfile::tempdir().unwrap();
        let logs = dir.path().join("logs");
        std::fs::create_dir_all(&logs).unwrap();
        std::fs::write(logs.join(LOG), "a log line\n").unwrap();
        let status = Status::Running {
            connection: Connection {
                host: "127.0.0.1".to_string(),
                port: 1,
                username: "synnax".to_string(),
                password: "launch-secret".to_string(),
            },
        };
        let history = History {
            starts: 2,
            ready_at: Some(5),
            last_exit: Some("exited with 3".to_string()),
        };
        let dest = dir.path().join("out.zip");
        export(
            &dest,
            "0.58.0",
            &status,
            &history,
            &dir.path().join("data"),
            &logs,
        )
        .unwrap();

        let mut archive = zip::ZipArchive::new(File::open(&dest).unwrap()).unwrap();
        let mut read = |name: &str| {
            let mut text = String::new();
            archive
                .by_name(name)
                .unwrap()
                .read_to_string(&mut text)
                .unwrap();
            text
        };
        assert_eq!(read("logs/core.log"), "a log line\n");
        let summary = read("diagnostics.json");
        assert!(!summary.contains("launch-secret"), "{summary}");
        let summary: serde_json::Value = serde_json::from_str(&summary).unwrap();
        assert_eq!(summary["state"]["state"], "running");
        assert_eq!(summary["version"], "0.58.0");
        assert_eq!(summary["history"]["starts"], 2);
        assert_eq!(summary["history"]["lastExit"], "exited with 3");
    }
}
