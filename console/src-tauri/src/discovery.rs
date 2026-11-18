// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! How scripts on this computer reach the embedded Core: its address in the discovery
//! file, and the API key in the OS keychain. The client libraries read the key through
//! the helper program that the discovery file names.

use std::collections::HashMap;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::process::ExitCode;

use keyring_core::{Entry, Error};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

/// The keychain service and account of the API key.
pub const KEY_SERVICE: &str = "com.synnaxlabs.desktop";
pub const KEY_ACCOUNT: &str = "api_key";

/// The file name of the program that prints the API key. Synnax Desktop installs it
/// next to its own executable.
pub const KEY_HELPER: &str = if cfg!(windows) {
    "synnax-desktop-key.exe"
} else {
    "synnax-desktop-key"
};

/// What the discovery file holds. It holds no secret.
#[derive(Debug, PartialEq, Serialize, Deserialize)]
pub struct File {
    pub host: String,
    pub port: u16,
    pub secure: bool,
    /// The absolute path of the program that prints the API key.
    pub api_key_helper: PathBuf,
}

/// The field where Synnax Desktop 1.1 saved the API key in the discovery file.
const LEGACY_KEY: &str = "api_key";

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
    // The uninstall docs name the Windows credential by this target name.
    let target = format!("{KEY_ACCOUNT}@{KEY_SERVICE}");
    if cfg!(windows) {
        modifiers.insert("target", target.as_str());
    }
    Entry::new_with_modifiers(KEY_SERVICE, KEY_ACCOUNT, &modifiers).map_err(io::Error::other)
}

/// Returns the API key that Synnax Desktop 1.1 saved in plaintext in the discovery file
/// at `path`, or `None` when the file holds none. The caller revokes it in the Core and
/// then calls [`forget_legacy_key`]. It never moves to the keychain.
pub fn legacy_key(path: &Path) -> io::Result<Option<String>> {
    Ok(read_file(path)?
        .and_then(|mut fields| fields.remove(LEGACY_KEY))
        .and_then(|key| key.as_str().map(str::to_string)))
}

/// Rewrites the discovery file at `path` without `api_key`, a key that Synnax Desktop
/// 1.1 saved in it. Leaves the file alone when it holds another key or none.
pub fn forget_legacy_key(path: &Path, api_key: &str) -> io::Result<()> {
    let Some(mut fields) = read_file(path)? else {
        return Ok(());
    };
    if fields.get(LEGACY_KEY).and_then(|key| key.as_str()) != Some(api_key) {
        return Ok(());
    }
    fields.remove(LEGACY_KEY);
    write(path, &fields)
}

fn read_file(path: &Path) -> io::Result<Option<Map<String, Value>>> {
    match std::fs::read(path) {
        Ok(bytes) => Ok(Some(serde_json::from_slice(&bytes)?)),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(err),
    }
}

/// Returns the API key saved in `key`, or `None` when there is none.
pub fn read_key(key: &Entry) -> io::Result<Option<String>> {
    match key.get_password() {
        Ok(secret) => Ok(Some(secret)),
        Err(Error::NoEntry) => Ok(None),
        Err(err) => Err(io::Error::other(err)),
    }
}

/// Writes the API key saved in `key` to `out`, followed by a newline, and returns
/// success. When `key` is an error, no key is saved, or the keychain fails, writes one
/// line to `err` and returns failure. Nothing written to `err` holds the key.
pub fn print_key(key: io::Result<Entry>, out: &mut impl Write, err: &mut impl Write) -> ExitCode {
    let printed = key.and_then(|key| read_key(&key)).and_then(|api_key| {
        let api_key = api_key.ok_or_else(|| {
            io::Error::other(
                "Synnax Desktop has saved no API key. Open Synnax Desktop to save one.",
            )
        })?;
        writeln!(out, "{api_key}")?;
        out.flush()
    });
    match printed {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            // A failed write to stderr leaves only the exit code to report the error.
            let _ = writeln!(err, "failed to read the Synnax Desktop API key: {error}");
            ExitCode::FAILURE
        }
    }
}

/// Saves `api_key` in the keychain and `file` to `path`, but only when the saved key is
/// `expected`, or, for `None`, when there is none. Callers must serialize their calls
/// and forget a legacy key first, because the write drops it from the file.
///
/// Returns whether it saved them.
pub fn swap(
    path: &Path,
    key: &Entry,
    file: &File,
    api_key: &str,
    expected: Option<&str>,
) -> io::Result<bool> {
    if read_key(key)?.as_deref() != expected {
        return Ok(false);
    }
    write(path, file)?;
    key.set_password(api_key).map_err(io::Error::other)?;
    Ok(true)
}

/// Writes `file` to `path`. The write goes to a temporary file first, so a reader never
/// sees half a file.
fn write(path: &Path, file: &impl Serialize) -> io::Result<()> {
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
            api_key_helper: PathBuf::from("/opt/synnax").join(KEY_HELPER),
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
        assert_eq!(read_key(&key).unwrap().as_deref(), Some("syk_secret"));
        assert_eq!(
            stored(&path),
            serde_json::json!({
                "host": "127.0.0.1",
                "port": 4321,
                "secure": false,
                "api_key_helper": file().api_key_helper,
            })
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
        assert_eq!(read_key(&key).unwrap().as_deref(), Some("syk_secret"));
        assert!(swap(&path, &key, &next, "syk_next", Some("syk_secret")).unwrap());
        assert_eq!(read_key(&key).unwrap().as_deref(), Some("syk_next"));
        assert_eq!(stored(&path)["port"], 5432);
        assert!(!path.with_extension("json.partial").exists());
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
    fn reads_a_legacy_key_without_moving_it_to_the_keychain() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write_legacy(&path);
        let key = key();
        assert_eq!(legacy_key(&path).unwrap().as_deref(), Some("syk_old"));
        assert_eq!(read_key(&key).unwrap(), None);
        assert_eq!(stored(&path)["api_key"], "syk_old");
    }

    #[test]
    fn forgets_only_the_given_legacy_key() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        write_legacy(&path);
        forget_legacy_key(&path, "syk_other").unwrap();
        assert_eq!(stored(&path)["api_key"], "syk_old");
        forget_legacy_key(&path, "syk_old").unwrap();
        assert_eq!(
            stored(&path),
            serde_json::json!({"host": "127.0.0.1", "port": 4321, "secure": false})
        );
        assert_eq!(legacy_key(&path).unwrap(), None);
        forget_legacy_key(&dir.path().join("missing.json"), "syk_old").unwrap();
    }

    /// Runs [`print_key`] and returns its exit code, stdout, and stderr.
    fn print(key: io::Result<Entry>) -> (ExitCode, String, String) {
        let (mut out, mut err) = (Vec::new(), Vec::new());
        let code = print_key(key, &mut out, &mut err);
        let text = |bytes| String::from_utf8(bytes).unwrap();
        (code, text(out), text(err))
    }

    #[test]
    fn prints_the_saved_key_and_a_newline() {
        let key = key();
        key.set_password("syk_secret").unwrap();
        let (code, out, err) = print(Ok(key));
        assert_eq!(code, ExitCode::SUCCESS);
        assert_eq!(out, "syk_secret\n");
        assert_eq!(err, "");
    }

    #[test]
    fn fails_with_one_line_when_no_key_is_saved() {
        let (code, out, err) = print(Ok(key()));
        assert_eq!(code, ExitCode::FAILURE);
        assert_eq!(out, "");
        assert_eq!(
            err,
            "failed to read the Synnax Desktop API key: Synnax Desktop has saved no API \
             key. Open Synnax Desktop to save one.\n"
        );
    }

    #[test]
    fn fails_with_one_line_when_there_is_no_keychain() {
        let (code, out, err) = print(Err(io::Error::other("no keychain")));
        assert_eq!(code, ExitCode::FAILURE);
        assert_eq!(out, "");
        assert_eq!(
            err,
            "failed to read the Synnax Desktop API key: no keychain\n"
        );
    }

    #[test]
    fn hides_the_key_when_the_keychain_fails() {
        let key = key();
        key.set_password("syk_secret").unwrap();
        let mock: &keyring_core::mock::Cred = key.as_any().downcast_ref().unwrap();
        mock.set_error(Error::NoStorageAccess("the keychain is locked".into()));
        let (code, out, err) = print(Ok(key));
        assert_eq!(code, ExitCode::FAILURE);
        assert_eq!(out, "");
        assert!(err.contains("the keychain is locked"));
        assert!(!err.contains("syk_secret"));
    }

    #[test]
    fn reads_no_key_when_nothing_is_saved() {
        let dir = tempfile::tempdir().unwrap();
        assert_eq!(legacy_key(&path(dir.path())).unwrap(), None);
        assert_eq!(read_key(&key()).unwrap(), None);
    }

    #[test]
    fn fails_to_read_a_malformed_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = path(dir.path());
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, "not json").unwrap();
        assert!(legacy_key(&path).is_err());
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
