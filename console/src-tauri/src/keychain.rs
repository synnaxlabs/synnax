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

use keyring::{Entry, Error};

/// The keychain service every Console token is stored under.
const SERVICE: &str = "com.synnaxlabs.console";

fn entry(account: &str) -> Result<Entry, String> {
    Entry::new(SERVICE, account).map_err(|e| e.to_string())
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
pub fn keychain_set(account: String, secret: String) -> Result<(), String> {
    entry(&account)?
        .set_password(&secret)
        .map_err(|e| e.to_string())
}

/// Removes the account's token. Removing one that is not stored is not an error.
#[tauri::command]
pub fn keychain_delete(account: String) -> Result<(), String> {
    match entry(&account)?.delete_credential() {
        Ok(()) | Err(Error::NoEntry) => Ok(()),
        Err(e) => Err(e.to_string()),
    }
}
