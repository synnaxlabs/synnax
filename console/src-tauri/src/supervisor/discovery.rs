// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! How scripts on this computer reach the embedded Core: its address in the discovery
//! file, and the API key in the OS keychain. The client libraries read both.

use std::collections::HashMap;
use std::io::{self, Write};
use std::path::{Path, PathBuf};

use keyring_core::{Entry, Error};
use serde::{Deserialize, Serialize};

/// The keychain service and account of the API key. The Python client reads the key
/// with these names.
pub const KEY_SERVICE: &str = "com.synnaxlabs.desktop";
pub const KEY_ACCOUNT: &str = "api_key";

/// What the discovery file holds. The fields match the options of the Python client.
#[derive(Debug, PartialEq, Serialize, Deserialize)]
pub struct File {
    pub host: String,
    pub port: u16,
    pub secure: bool,
}

/// The discovery file as read. Synnax Desktop 1.1 also wrote the API key in it.
#[derive(Deserialize)]
struct Legacy {
    #[serde(flatten)]
    file: File,
    api_key: Option<String>,
}

/// Returns the path of the discovery file in the given home directory.
pub fn path(home: &Path) -> PathBuf {
    home.join(".synnax").join("desktop.json")
}

/// Returns the keychain entry of the API key.
pub fn key_entry() -> io::Result<Entry> {
    keyring::Entry::store_status()
        .as_ref()
        .map_err(|err| io::Error::other(err.to_string()))?;
    let mut modifiers = HashMap::new();
    // The Python keyring reads a Windows credential by this target name.
    let target = format!("{KEY_ACCOUNT}@{KEY_SERVICE}");
    if cfg!(windows) {
        modifiers.insert("target", target.as_str());
    }
    Entry::new_with_modifiers(KEY_SERVICE, KEY_ACCOUNT, &modifiers).map_err(io::Error::other)
}

/// Returns the keychain entry of the API key. When this session has no keychain,
/// first removes a key that an older version left in the discovery file at `path`, so
/// the key never stays on disk in plaintext.
pub fn key_entry_for(path: &Path) -> io::Result<Entry> {
    key_entry().or_else(|err| {
        drop_legacy_key(path)?;
        Err(err)
    })
}

/// Rewrites the discovery file at `path` without the API key that an older version
/// saved in it. Does nothing when the file holds no key.
fn drop_legacy_key(path: &Path) -> io::Result<()> {
    if let Some(Legacy {
        file,
        api_key: Some(_),
    }) = read_file(path)?
    {
        write(path, &file)?;
    }
    Ok(())
}

fn read_file(path: &Path) -> io::Result<Option<Legacy>> {
    match std::fs::read(path) {
        Ok(bytes) => Ok(Some(serde_json::from_slice(&bytes)?)),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(err),
    }
}

fn get_key(key: &Entry) -> io::Result<Option<String>> {
    match key.get_password() {
        Ok(secret) => Ok(Some(secret)),
        Err(Error::NoEntry) => Ok(None),
        Err(err) => Err(io::Error::other(err)),
    }
}

/// Returns the saved API key, or `None` when there is none. Moves a key that an older
/// version saved in the discovery file at `path` into the keychain. Callers must
/// serialize their calls.
pub fn read_key(path: &Path, key: &Entry) -> io::Result<Option<String>> {
    if let Some(Legacy {
        file,
        api_key: Some(api_key),
    }) = read_file(path)?
    {
        let moved = match get_key(key) {
            Ok(Some(_)) => Ok(()),
            Ok(None) => key.set_password(&api_key).map_err(io::Error::other),
            Err(err) => Err(err),
        };
        // The plaintext key leaves the file even when the keychain refuses it.
        write(path, &file)?;
        moved?;
    }
    get_key(key)
}

/// Saves `api_key` in the keychain and `file` to `path`, but only when the saved key is
/// `expected`, or, for `None`, when there is none. Callers must serialize their calls.
///
/// Returns whether it saved them.
pub fn swap(
    path: &Path,
    key: &Entry,
    file: &File,
    api_key: &str,
    expected: Option<&str>,
) -> io::Result<bool> {
    if read_key(path, key)?.as_deref() != expected {
        return Ok(false);
    }
    // The file goes first: a failed keychain write then leaves the expected key saved.
    write(path, file)?;
    key.set_password(api_key).map_err(io::Error::other)?;
    Ok(true)
}

/// Writes `file` to `path`. The write goes to a temporary file first, so a reader never
/// sees half a file.
fn write(path: &Path, file: &File) -> io::Result<()> {
    let dir = path
        .parent()
        .ok_or_else(|| io::Error::other("the discovery file has no parent directory"))?;
    std::fs::create_dir_all(dir)?;
    let partial = path.with_extension("json.partial");
    let mut out = std::fs::File::create(&partial)?;
    out.write_all(&serde_json::to_vec_pretty(file)?)?;
    out.sync_all()?;
    std::fs::rename(partial, path)
}

/// Removes the discovery file at `path`, if it exists.
pub fn remove_file(path: &Path) -> io::Result<()> {
    match std::fs::remove_file(path) {
        Err(err) if err.kind() != io::ErrorKind::NotFound => Err(err),
        _ => Ok(()),
    }
}

/// Deletes the API key from the keychain, if it exists.
pub fn remove_key(key: &Entry) -> io::Result<()> {
    match key.delete_credential() {
        Ok(()) | Err(Error::NoEntry) => Ok(()),
        Err(err) => Err(io::Error::other(err)),
    }
}

#[cfg(test)]
mod tests {
    use keyring_core::api::CredentialStoreApi;

    use super::*;

    fn file() -> File {
        File {
            host: "127.0.0.1".to_string(),
            port: 4321,
            secure: false,
        }
    }

    fn key() -> Entry {
        keyring_core::mock::Store::new()
            .unwrap()
            .build(KEY_SERVICE, KEY_ACCOUNT, None)
            .unwrap()
    }

    fn stored(path: &Path) -> serde_json::Value {
        serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
    }

    #[test]
    fn places_the_file_in_the_synnax_directory_of_the_home() {
        let home = Path::new("home");
        assert_eq!(path(home), home.join(".synnax").join("desktop.json"));
    }

    #[test]
    fn saves_the_key_in_the_keychain_and_the_address_in_the_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        let key = key();
        assert!(swap(&path, &key, &file(), "syk_secret", None).unwrap());
        assert_eq!(key.get_password().unwrap(), "syk_secret");
        assert_eq!(
            read_key(&path, &key).unwrap().as_deref(),
            Some("syk_secret")
        );
        assert_eq!(
            stored(&path),
            serde_json::json!({"host": "127.0.0.1", "port": 4321, "secure": false})
        );
    }

    #[test]
    fn swaps_only_when_the_keychain_holds_the_expected_key() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        let key = key();
        assert!(!swap(&path, &key, &file(), "syk_secret", Some("syk_other")).unwrap());
        assert!(!path.exists());
        assert!(swap(&path, &key, &file(), "syk_secret", None).unwrap());
        let next = File {
            port: 5432,
            ..file()
        };
        assert!(!swap(&path, &key, &next, "syk_next", None).unwrap());
        assert!(!swap(&path, &key, &next, "syk_next", Some("syk_other")).unwrap());
        assert_eq!(
            read_key(&path, &key).unwrap().as_deref(),
            Some("syk_secret")
        );
        assert!(swap(&path, &key, &next, "syk_next", Some("syk_secret")).unwrap());
        assert_eq!(read_key(&path, &key).unwrap().as_deref(), Some("syk_next"));
        assert_eq!(stored(&path)["port"], 5432);
        assert!(!path.with_extension("json.partial").exists());
    }

    #[test]
    fn moves_a_key_from_an_older_file_into_the_keychain() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(
            &path,
            r#"{"host":"127.0.0.1","port":4321,"secure":false,"api_key":"syk_old"}"#,
        )
        .unwrap();
        let key = key();
        assert_eq!(read_key(&path, &key).unwrap().as_deref(), Some("syk_old"));
        assert_eq!(key.get_password().unwrap(), "syk_old");
        assert_eq!(
            stored(&path),
            serde_json::json!({"host": "127.0.0.1", "port": 4321, "secure": false})
        );
    }

    #[test]
    fn keeps_the_keychain_key_over_one_in_an_older_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(
            &path,
            r#"{"host":"127.0.0.1","port":4321,"secure":false,"api_key":"syk_old"}"#,
        )
        .unwrap();
        let key = key();
        key.set_password("syk_new").unwrap();
        assert_eq!(read_key(&path, &key).unwrap().as_deref(), Some("syk_new"));
        assert!(stored(&path).get("api_key").is_none());
    }

    fn write_legacy(path: &Path) {
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(
            path,
            r#"{"host":"127.0.0.1","port":4321,"secure":false,"api_key":"syk_old"}"#,
        )
        .unwrap();
    }

    #[test]
    fn drops_an_older_files_key_when_the_keychain_fails() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write_legacy(&path);
        let key = key();
        let mock: &keyring_core::mock::Cred = key.as_any().downcast_ref().unwrap();
        mock.set_error(Error::NoStorageAccess(io::Error::other("locked").into()));
        assert!(read_key(&path, &key).is_err());
        assert!(stored(&path).get("api_key").is_none());
    }

    #[test]
    fn drops_an_older_files_key_without_a_keychain() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write_legacy(&path);
        drop_legacy_key(&path).unwrap();
        assert_eq!(
            stored(&path),
            serde_json::json!({"host": "127.0.0.1", "port": 4321, "secure": false})
        );
        drop_legacy_key(&path).unwrap();
    }

    #[test]
    fn reads_no_key_when_nothing_is_saved() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(read_key(&path(dir.path()), &key()).unwrap(), None);
    }

    #[test]
    fn fails_to_read_a_malformed_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, "not json").unwrap();
        assert!(read_key(&path, &key()).is_err());
    }

    #[test]
    fn removes_the_file_and_the_key_and_accepts_missing_ones() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        let key = key();
        assert!(swap(&path, &key, &file(), "syk_secret", None).unwrap());
        remove_file(&path).unwrap();
        remove_key(&key).unwrap();
        assert!(!path.exists());
        assert!(matches!(key.get_password(), Err(Error::NoEntry)));
        remove_file(&path).unwrap();
        remove_key(&key).unwrap();
    }
}
