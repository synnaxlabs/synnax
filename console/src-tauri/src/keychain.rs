// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Session tokens for the Cores the Console connects to, kept in the operating
//! system's keychain so they never reach the Console's stored state.

use std::sync::Mutex;

use keyring::{Entry, Error};
use tauri::State;

/// The keychain service every Console token is stored under.
const SERVICE: &str = "com.synnaxlabs.console";

/// Serializes the writes of every window, so a delete that compares the stored token
/// first runs as one step.
#[derive(Default)]
pub struct Lock(Mutex<()>);

fn entry(account: &str) -> Result<Entry, String> {
    Entry::new(SERVICE, account).map_err(|e| e.to_string())
}

fn delete(entry: &Entry, expected: Option<&str>) -> Result<(), String> {
    if let Some(expected) = expected {
        match entry.get_password() {
            Ok(token) if token == expected => {}
            Ok(_) | Err(Error::NoEntry) => return Ok(()),
            Err(e) => return Err(e.to_string()),
        }
    }
    match entry.delete_credential() {
        Ok(()) | Err(Error::NoEntry) => Ok(()),
        Err(e) => Err(e.to_string()),
    }
}

/// Returns the token stored for the account, or None when there is none.
#[tauri::command]
pub fn keychain_get(account: String) -> Result<Option<String>, String> {
    match entry(&account)?.get_password() {
        Ok(token) => Ok(Some(token)),
        Err(Error::NoEntry) => Ok(None),
        Err(e) => Err(e.to_string()),
    }
}

/// Stores the token for the account, replacing any earlier one.
#[tauri::command]
pub fn keychain_set(lock: State<'_, Lock>, account: String, secret: String) -> Result<(), String> {
    let _held = lock.0.lock().map_err(|e| e.to_string())?;
    entry(&account)?
        .set_password(&secret)
        .map_err(|e| e.to_string())
}

/// Removes the account's token. Given `expected`, removes it only while the keychain
/// still holds that token, so a token another window saved since stays. Removing one
/// that is not stored is not an error.
#[tauri::command]
pub fn keychain_delete(
    lock: State<'_, Lock>,
    account: String,
    expected: Option<String>,
) -> Result<(), String> {
    let _held = lock.0.lock().map_err(|e| e.to_string())?;
    delete(&entry(&account)?, expected.as_deref())
}

#[cfg(test)]
mod tests {
    use keyring_core::api::CredentialStoreApi;

    use super::*;

    fn stored(token: &str) -> Entry {
        let inner = keyring_core::mock::Store::new()
            .unwrap()
            .build(SERVICE, "account", None)
            .unwrap();
        let entry = Entry { inner };
        entry.set_password(token).unwrap();
        entry
    }

    #[test]
    fn deletes_the_token_it_expects() {
        let entry = stored("sys_old");
        delete(&entry, Some("sys_old")).unwrap();
        assert!(matches!(entry.get_password(), Err(Error::NoEntry)));
    }

    #[test]
    fn keeps_a_token_saved_since() {
        let entry = stored("sys_new");
        delete(&entry, Some("sys_old")).unwrap();
        assert_eq!(entry.get_password().unwrap(), "sys_new");
    }

    #[test]
    fn deletes_any_token_when_none_is_expected() {
        let entry = stored("sys_new");
        delete(&entry, None).unwrap();
        assert!(matches!(entry.get_password(), Err(Error::NoEntry)));
    }

    #[test]
    fn ignores_a_missing_token() {
        let entry = stored("sys_old");
        entry.delete_credential().unwrap();
        delete(&entry, Some("sys_old")).unwrap();
        delete(&entry, None).unwrap();
    }
}
