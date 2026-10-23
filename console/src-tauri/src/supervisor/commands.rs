// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Binds the supervisor to the Tauri app: its commands, its status event, and its
//! place in the app lifecycle.

use std::path::{Path, PathBuf};

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager, Runtime, State};

use super::{Config, History, Status, Supervisor, diagnostics};
use crate::install;

/// The event every window receives on each status change.
const STATUS_EVENT: &str = "supervisor://status";

/// The name of the bundled Core executable, as `bundle.externalBin` installs it.
const PROGRAM: &str = if cfg!(windows) {
    "synnax-core.exe"
} else {
    "synnax-core"
};

/// What the diagnostics commands read, managed as app state.
pub struct Paths {
    version: String,
    data_dir: PathBuf,
    log_dir: PathBuf,
}

/// What the diagnostics dialog shows beside the status.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Diagnostics {
    version: String,
    history: History,
    data_dir: PathBuf,
    log_dir: PathBuf,
    /// The size of the data directory in bytes.
    data_size: u64,
}

/// Starts the supervisor, manages it as app state, and forwards its status to every
/// window.
pub fn init<R: Runtime>(app: &AppHandle<R>) -> Result<(), Box<dyn std::error::Error>> {
    // Tauri installs a sidecar next to the app executable.
    let program = tauri::utils::platform::current_exe()?
        .parent()
        .ok_or("the app executable has no parent directory")?
        .join(PROGRAM);
    let local = app.path().app_local_data_dir()?;
    let log_dir = app.path().app_log_dir()?;
    let version = app.package_info().version.to_string();
    let data_dir = local.join("core");
    let cfg = Config::new(
        program,
        version.clone(),
        local,
        data_dir.clone(),
        log_dir.clone(),
    );
    // Supervisor::open spawns onto the current runtime, which setup does not enter.
    let supervisor = tauri::async_runtime::block_on(async { Supervisor::open(cfg) })?;
    let mut status = supervisor.subscribe();
    let handle = app.clone();
    tauri::async_runtime::spawn(async move {
        while status.changed().await.is_ok() {
            let current = status.borrow_and_update().clone();
            if let Err(err) = handle.emit(STATUS_EVENT, current) {
                eprintln!("failed to emit the supervisor status: {err}");
            }
        }
    });
    app.manage(supervisor);
    app.manage(Paths {
        version,
        data_dir,
        log_dir,
    });
    Ok(())
}

/// Stops the Core and returns once its process has exited. Call it before the app
/// exits. It is safe to call from a thread of the async runtime.
pub fn shutdown<R: Runtime>(app: &AppHandle<R>) {
    let Some(supervisor) = app.try_state::<Supervisor>() else {
        return;
    };
    // block_on panics on a runtime thread, so a plain thread does the wait.
    std::thread::scope(|scope| {
        scope.spawn(|| tauri::async_runtime::block_on(supervisor.stop()));
    });
}

#[tauri::command]
pub fn supervisor_status(supervisor: State<'_, Supervisor>) -> Status {
    supervisor.status()
}

#[tauri::command]
pub fn supervisor_history(supervisor: State<'_, Supervisor>) -> History {
    supervisor.history()
}

#[tauri::command]
pub async fn supervisor_restart(supervisor: State<'_, Supervisor>) -> Result<(), ()> {
    supervisor.restart().await;
    Ok(())
}

#[tauri::command]
pub async fn supervisor_stop(supervisor: State<'_, Supervisor>) -> Result<(), ()> {
    supervisor.stop().await;
    Ok(())
}

/// Erases everything the Core stored, then starts the app again. A new launch is the
/// one state in which no window holds data of the erased Core.
#[tauri::command]
pub async fn supervisor_reset<R: Runtime>(
    app: AppHandle<R>,
    supervisor: State<'_, Supervisor>,
    paths: State<'_, Paths>,
) -> Result<(), String> {
    let data_dir = paths.data_dir.clone();
    let local = app
        .path()
        .app_local_data_dir()
        .map_err(std::io::Error::other);
    // The record only feeds analytics, so a failure costs the report and never the reset
    // that recovers a broken Core.
    if let Err(err) = blocking(move || record_reset(&local?, &data_dir)).await {
        eprintln!("failed to record the reset: {err}");
    }
    supervisor.reset().await.map_err(|err| err.to_string())?;
    app.restart()
}

/// Stores the size of the data a reset is about to erase, for the next launch to report.
fn record_reset(local: &Path, data_dir: &Path) -> std::io::Result<()> {
    install::record_reset(local, diagnostics::dir_size(data_dir)?)
}

#[tauri::command]
pub async fn supervisor_diagnostics(
    supervisor: State<'_, Supervisor>,
    paths: State<'_, Paths>,
) -> Result<Diagnostics, String> {
    let data_dir = paths.data_dir.clone();
    let data_size = blocking(move || diagnostics::dir_size(&data_dir)).await?;
    Ok(Diagnostics {
        version: paths.version.clone(),
        history: supervisor.history(),
        data_dir: paths.data_dir.clone(),
        log_dir: paths.log_dir.clone(),
        data_size,
    })
}

/// Returns the last lines of the Core log.
#[tauri::command]
pub async fn supervisor_log_tail(paths: State<'_, Paths>) -> Result<String, String> {
    let log_dir = paths.log_dir.clone();
    blocking(move || diagnostics::log_tail(&log_dir)).await
}

/// Writes a diagnostics archive for support to `path`.
#[tauri::command]
pub async fn supervisor_export_diagnostics(
    path: PathBuf,
    supervisor: State<'_, Supervisor>,
    paths: State<'_, Paths>,
    install: State<'_, install::Record>,
) -> Result<(), String> {
    let (status, history) = (supervisor.status(), supervisor.history());
    let (version, data_dir, log_dir) = (
        paths.version.clone(),
        paths.data_dir.clone(),
        paths.log_dir.clone(),
    );
    let id = install.0.as_ref().ok().map(|info| info.id.clone());
    blocking(move || {
        diagnostics::export(
            &path,
            &version,
            id.as_deref(),
            &status,
            &history,
            &data_dir,
            &log_dir,
        )
    })
    .await
}

/// Opens the log directory in the platform file manager.
#[tauri::command]
pub fn supervisor_show_logs(paths: State<'_, Paths>) -> Result<(), String> {
    reveal(&paths.log_dir)
}

/// Opens the data directory in the platform file manager.
#[tauri::command]
pub fn supervisor_show_data(paths: State<'_, Paths>) -> Result<(), String> {
    reveal(&paths.data_dir)
}

/// Runs file work off the async runtime and flattens its errors to a message.
async fn blocking<T: Send + 'static>(
    work: impl FnOnce() -> std::io::Result<T> + Send + 'static,
) -> Result<T, String> {
    tauri::async_runtime::spawn_blocking(work)
        .await
        .map_err(|err| err.to_string())?
        .map_err(|err| err.to_string())
}

fn reveal(dir: &Path) -> Result<(), String> {
    let opener = if cfg!(target_os = "macos") {
        "open"
    } else if cfg!(windows) {
        "explorer"
    } else {
        "xdg-open"
    };
    std::process::Command::new(opener)
        .arg(dir)
        .spawn()
        .map(|_| ())
        .map_err(|err| format!("failed to open {}: {err}", dir.display()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn records_the_size_of_the_data_a_reset_erases() {
        let local = tempfile::tempdir().unwrap();
        let data = tempfile::tempdir().unwrap();
        std::fs::write(
            local.path().join("install.json"),
            r#"{"id":"abc","launched_at":1}"#,
        )
        .unwrap();
        std::fs::write(data.path().join("segment"), [0u8; 10]).unwrap();
        record_reset(local.path(), data.path()).unwrap();
        let stored: serde_json::Value =
            serde_json::from_slice(&std::fs::read(local.path().join("install.json")).unwrap())
                .unwrap();
        assert_eq!(stored["erased_bytes"], 10);
    }

    #[test]
    fn fails_to_record_a_reset_without_an_install_record() {
        let local = tempfile::tempdir().unwrap();
        let data = tempfile::tempdir().unwrap();
        let err = record_reset(local.path(), data.path()).unwrap_err();
        assert_eq!(err.to_string(), "no install record");
    }
}
