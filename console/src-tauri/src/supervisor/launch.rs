// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! The inputs of one Core launch: its credentials, port, and config file.

use std::io;
use std::net::{Ipv4Addr, TcpListener};
use std::path::Path;

use serde_json::json;

/// The host the Core binds and clients dial. An IP, because `localhost` can resolve to
/// the IPv6 loopback, which the Core does not bind.
pub const HOST: Ipv4Addr = Ipv4Addr::LOCALHOST;

/// The root username of the embedded Core.
pub const USERNAME: &str = "synnax";

/// The environment variable the Core reads its root password from.
pub const PASSWORD_ENV: &str = "SYNNAX_PASSWORD";

/// Returns a new random root password: 32 bytes, hex encoded.
pub fn password() -> io::Result<String> {
    let mut bytes = [0u8; 32];
    getrandom::fill(&mut bytes).map_err(|err| io::Error::other(err.to_string()))?;
    Ok(bytes.iter().map(|b| format!("{b:02x}")).collect())
}

/// Returns a TCP port that is free on the loopback interface. Another process can take
/// the port before the Core binds it; the Core then exits and the supervisor restarts
/// it on a new port.
pub fn free_port() -> io::Result<u16> {
    Ok(TcpListener::bind((HOST, 0))?.local_addr()?.port())
}

/// Returns `preferred` when it is still free on the loopback interface, and a new free
/// port when it is not.
pub fn port(preferred: Option<u16>) -> io::Result<u16> {
    match preferred {
        Some(port) if TcpListener::bind((HOST, port)).is_ok() => Ok(port),
        _ => free_port(),
    }
}

/// Writes the Core config file to `path`. The file holds no secret: the password
/// travels in [`PASSWORD_ENV`].
pub fn write_config(path: &Path, port: u16, data_dir: &Path, log_dir: &Path) -> io::Result<()> {
    let config = json!({
        "listen": [{ "address": format!("{HOST}:{port}"), "loopback": true }],
        "insecure": true,
        "username": USERNAME,
        "data": data_dir,
        "log-file-path": log_dir.join("core.log"),
        "stop-on-stdin-close": true,
    });
    std::fs::write(path, serde_json::to_vec_pretty(&config)?)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn makes_a_distinct_64_character_hex_password_per_call() {
        let (a, b) = (password().unwrap(), password().unwrap());
        assert_eq!(a.len(), 64);
        assert!(a.chars().all(|c| c.is_ascii_hexdigit()));
        assert_ne!(a, b);
    }

    #[test]
    fn picks_a_port_that_can_be_bound() {
        let port = free_port().unwrap();
        assert!(TcpListener::bind((HOST, port)).is_ok());
    }

    #[test]
    fn keeps_a_preferred_port_that_is_free() {
        let preferred = free_port().unwrap();
        assert_eq!(port(Some(preferred)).unwrap(), preferred);
    }

    #[test]
    fn replaces_a_preferred_port_that_is_taken() {
        let taken = TcpListener::bind((HOST, 0)).unwrap();
        let preferred = taken.local_addr().unwrap().port();
        assert_ne!(port(Some(preferred)).unwrap(), preferred);
    }

    #[test]
    fn writes_a_loopback_config_without_a_password() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("core.json");
        let data = dir.path().join("data dir");
        write_config(&path, 4321, &data, dir.path()).unwrap();
        let text = std::fs::read_to_string(&path).unwrap();
        let config: serde_json::Value = serde_json::from_str(&text).unwrap();
        assert_eq!(
            config["listen"],
            json!([{ "address": "127.0.0.1:4321", "loopback": true }])
        );
        assert_eq!(config["insecure"], json!(true));
        assert_eq!(config["stop-on-stdin-close"], json!(true));
        assert_eq!(config["username"], json!(USERNAME));
        assert_eq!(config["data"], json!(data));
        assert_eq!(config["log-file-path"], json!(dir.path().join("core.log")));
        assert!(!text.contains("password"));
    }
}
