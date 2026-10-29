// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Reads and sets the star the user gives the Synnax repository on GitHub. The calls go
//! through the GitHub CLI, which holds the credentials, so the app never handles them.

use std::io;
use std::process::{Output, Stdio};
use std::time::Duration;

use serde::Serialize;
use tokio::process::Command;

const PATH: &str = "user/starred/synnaxlabs/synnax";

const STATUS_ARGS: [&str; 3] = ["api", "--silent", PATH];

const STAR_ARGS: [&str; 5] = ["api", "--silent", "--method", "PUT", PATH];

/// Covers the startup of the login shell, which loads the user's profile.
const TIMEOUT: Duration = Duration::from_secs(10);

#[derive(Debug, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum Status {
    Starred,
    NotStarred,
    /// The CLI is missing, is not signed in, or did not get an answer from GitHub.
    Unknown,
}

#[tauri::command]
pub async fn github_star_status() -> Status {
    match run(&STATUS_ARGS).await {
        Ok(output) => read_status(output.status.success(), &output.stderr),
        Err(_) => Status::Unknown,
    }
}

/// Returns true when GitHub recorded the star.
#[tauri::command]
pub async fn github_star() -> bool {
    run(&STAR_ARGS)
        .await
        .is_ok_and(|output| output.status.success())
}

fn read_status(success: bool, stderr: &[u8]) -> Status {
    if success {
        Status::Starred
    } else if String::from_utf8_lossy(stderr).contains("(HTTP 404)") {
        Status::NotStarred
    } else {
        Status::Unknown
    }
}

async fn run(args: &[&str]) -> io::Result<Output> {
    let child = command(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::piped())
        .kill_on_drop(true)
        .spawn()?;
    tokio::time::timeout(TIMEOUT, child.wait_with_output()).await?
}

/// An app opened from the desktop inherits a minimal `PATH` that misses a CLI installed
/// by Homebrew, so the call runs in the user's login shell.
#[cfg(unix)]
fn command(args: &[&str]) -> Command {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/sh".into());
    let mut cmd = Command::new(shell);
    cmd.arg("-ilc").arg(format!("gh {}", args.join(" ")));
    cmd
}

#[cfg(windows)]
fn command(args: &[&str]) -> Command {
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    let mut cmd = Command::new("gh");
    cmd.args(args).creation_flags(CREATE_NO_WINDOW);
    cmd
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reads_a_success_as_starred() {
        assert_eq!(read_status(true, b""), Status::Starred);
    }

    #[test]
    fn reads_a_not_found_as_not_starred() {
        assert_eq!(
            read_status(false, b"gh: Not Found (HTTP 404)\n"),
            Status::NotStarred
        );
    }

    #[test]
    fn reads_any_other_failure_as_unknown() {
        assert_eq!(
            read_status(
                false,
                b"To get started with GitHub CLI, run: gh auth login\n"
            ),
            Status::Unknown
        );
        assert_eq!(
            read_status(false, b"zsh: command not found: gh\n"),
            Status::Unknown
        );
    }
}
