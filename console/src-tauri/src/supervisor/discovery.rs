// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! The discovery file: the address and API key that let scripts on this computer reach
//! the embedded Core. The client libraries read it from the same path.

use std::io::{self, Write};
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

/// What the discovery file holds. The fields match the options of the Python client.
#[derive(Debug, PartialEq, Serialize, Deserialize)]
pub struct File {
    pub host: String,
    pub port: u16,
    pub secure: bool,
    pub api_key: String,
}

/// Returns the path of the discovery file in the given home directory.
pub fn path(home: &Path) -> PathBuf {
    home.join(".synnax").join("desktop.json")
}

/// Reads the discovery file at `path`, or returns `None` when there is none.
pub fn read(path: &Path) -> io::Result<Option<File>> {
    match std::fs::read(path) {
        Ok(bytes) => Ok(Some(serde_json::from_slice(&bytes)?)),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(err),
    }
}

/// Writes `file` to `path` so that only the current user can read it. The write goes to
/// a temporary file first, so a reader never sees half a file.
pub fn write(path: &Path, file: &File) -> io::Result<()> {
    let dir = path
        .parent()
        .ok_or_else(|| io::Error::other("the discovery file has no parent directory"))?;
    std::fs::create_dir_all(dir)?;
    let partial = path.with_extension("json.partial");
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create(true).truncate(true);
    // A Windows user profile is private to its user, so the default access applies.
    #[cfg(unix)]
    std::os::unix::fs::OpenOptionsExt::mode(&mut options, 0o600);
    let mut out = options.open(&partial)?;
    out.write_all(&serde_json::to_vec_pretty(file)?)?;
    out.sync_all()?;
    std::fs::rename(partial, path)
}

/// Removes the discovery file at `path`, if there is one.
pub fn remove(path: &Path) -> io::Result<()> {
    match std::fs::remove_file(path) {
        Err(err) if err.kind() != io::ErrorKind::NotFound => Err(err),
        _ => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn file() -> File {
        File {
            host: "127.0.0.1".to_string(),
            port: 4321,
            secure: false,
            api_key: "syk_key_secret".to_string(),
        }
    }

    #[test]
    fn places_the_file_in_the_synnax_directory_of_the_home() {
        let home = Path::new("home");
        assert_eq!(path(home), home.join(".synnax").join("desktop.json"));
    }

    #[test]
    fn reads_back_what_it_writes() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write(&path, &file()).unwrap();
        assert_eq!(read(&path).unwrap(), Some(file()));
    }

    #[test]
    fn writes_the_field_names_the_python_client_reads() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write(&path, &file()).unwrap();
        let stored: serde_json::Value =
            serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        assert_eq!(
            stored,
            serde_json::json!({
                "host": "127.0.0.1",
                "port": 4321,
                "secure": false,
                "api_key": "syk_key_secret",
            })
        );
    }

    #[cfg(unix)]
    #[test]
    fn lets_only_the_current_user_read_the_file() {
        use std::os::unix::fs::PermissionsExt;
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write(&path, &file()).unwrap();
        let mode = std::fs::metadata(&path).unwrap().permissions().mode();
        assert_eq!(mode & 0o777, 0o600);
    }

    #[test]
    fn replaces_an_existing_file_and_leaves_no_partial_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write(&path, &file()).unwrap();
        let next = File {
            port: 5432,
            ..file()
        };
        write(&path, &next).unwrap();
        assert_eq!(read(&path).unwrap(), Some(next));
        assert!(!path.with_extension("json.partial").exists());
    }

    #[test]
    fn reads_no_file_when_there_is_none() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(read(&path(dir.path())).unwrap(), None);
    }

    #[test]
    fn fails_to_read_a_malformed_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, "not json").unwrap();
        assert!(read(&path).is_err());
    }

    #[test]
    fn removes_the_file_and_accepts_a_missing_one() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write(&path, &file()).unwrap();
        remove(&path).unwrap();
        assert!(!path.exists());
        remove(&path).unwrap();
    }
}
