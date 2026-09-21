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

use std::path::PathBuf;

use tauri::{AppHandle, Emitter, Manager, Runtime, State};

use super::{Config, Status, Supervisor};

/// The event every window receives on each status change.
const STATUS_EVENT: &str = "supervisor://status";

/// The name of the bundled Core executable, as `bundle.externalBin` installs it.
const PROGRAM: &str = if cfg!(windows) {
    "synnax-core.exe"
} else {
    "synnax-core"
};

/// The log directory, managed so the show-logs command can reach it.
pub struct LogDir(PathBuf);

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
    let cfg = Config::new(program, local.clone(), local.join("core"), log_dir.clone());
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
    app.manage(LogDir(log_dir));
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
pub async fn supervisor_restart(supervisor: State<'_, Supervisor>) -> Result<(), ()> {
    supervisor.restart().await;
    Ok(())
}

#[tauri::command]
pub async fn supervisor_stop(supervisor: State<'_, Supervisor>) -> Result<(), ()> {
    supervisor.stop().await;
    Ok(())
}

/// Opens the log directory in the platform file manager.
#[tauri::command]
pub fn supervisor_show_logs(log_dir: State<'_, LogDir>) -> Result<(), String> {
    let opener = if cfg!(target_os = "macos") {
        "open"
    } else if cfg!(windows) {
        "explorer"
    } else {
        "xdg-open"
    };
    std::process::Command::new(opener)
        .arg(&log_dir.0)
        .spawn()
        .map(|_| ())
        .map_err(|err| format!("failed to open {}: {err}", log_dir.0.display()))
}
