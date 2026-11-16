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
use std::sync::atomic::{AtomicBool, Ordering};

use serde::Serialize;
use tauri::{AppHandle, Emitter, ExitRequestApi, Manager, RESTART_EXIT_CODE, Runtime, State};
use tokio::sync::Mutex;

use super::{Config, History, Status, Supervisor, diagnostics, discovery};
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

/// The discovery file, managed as app state. Each window of the app writes it.
pub struct Discovery {
    path: PathBuf,
    /// Makes the compare and the write of each save one step.
    lock: Mutex<()>,
}

/// The exit of the app, managed as app state. The app exits only once its Core has
/// stopped, so the next launch finds the data directory free.
#[derive(Default)]
struct Quit {
    /// Set once the stop for the exit starts.
    started: AtomicBool,
    /// Set once the Core has stopped.
    done: AtomicBool,
    /// Set when the app is launched again during the stop.
    relaunched: AtomicBool,
}

/// What a request to exit the app does.
#[derive(Debug, PartialEq)]
enum Decision {
    /// Hold the exit and stop the Core.
    Stop,
    /// Hold the exit until the stop that runs ends.
    Hold,
    /// Restart the app instead of the exit.
    Restart,
    /// Let the exit go ahead.
    Exit,
}

impl Quit {
    /// Starts the stop for the exit. Returns false when it has already started.
    fn start(&self) -> bool {
        !self.started.swap(true, Ordering::SeqCst)
    }

    fn finish(&self) {
        self.done.store(true, Ordering::SeqCst);
    }

    fn relaunch(&self) {
        if self.started.load(Ordering::SeqCst) {
            self.relaunched.store(true, Ordering::SeqCst);
        }
    }

    fn decide(&self, code: Option<i32>) -> Decision {
        if !self.done.load(Ordering::SeqCst) {
            return if self.start() {
                Decision::Stop
            } else {
                Decision::Hold
            };
        }
        if self.relaunched.load(Ordering::SeqCst) && code != Some(RESTART_EXIT_CODE) {
            return Decision::Restart;
        }
        Decision::Exit
    }
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
    let discovery_path = discovery::path(&app.path().home_dir()?);
    let cfg = Config::new(program, local, data_dir.clone(), log_dir.clone());
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
    app.manage(Discovery {
        path: discovery_path,
        lock: Mutex::new(()),
    });
    app.manage(Quit::default());
    Ok(())
}

/// Stops the Core, then exits the app with `code`. The exit request that follows goes
/// ahead, or restarts the app when it was launched again during the stop.
async fn stop_and_exit<R: Runtime>(app: AppHandle<R>, code: Option<i32>) {
    app.state::<Supervisor>().stop_for_exit().await;
    app.state::<Quit>().finish();
    match code {
        Some(RESTART_EXIT_CODE) => app.request_restart(),
        code => app.exit(code.unwrap_or(0)),
    }
}

/// Holds an exit of the app until its Core has stopped. Call it on each
/// `RunEvent::ExitRequested`.
pub fn exit_requested<R: Runtime>(app: &AppHandle<R>, code: Option<i32>, api: &ExitRequestApi) {
    match app.state::<Quit>().decide(code) {
        Decision::Stop => {
            api.prevent_exit();
            tauri::async_runtime::spawn(stop_and_exit(app.clone(), code));
        }
        Decision::Hold => api.prevent_exit(),
        Decision::Restart => {
            api.prevent_exit();
            app.request_restart();
        }
        Decision::Exit => {}
    }
}

/// Records a second launch of the app. One that arrives while the app exits restarts it.
pub fn relaunched<R: Runtime>(app: &AppHandle<R>) {
    app.state::<Quit>().relaunch();
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

/// Stops the Core while the windows stay open, then exits the app.
#[tauri::command]
pub async fn supervisor_quit<R: Runtime>(app: AppHandle<R>) -> Result<(), ()> {
    if app.state::<Quit>().start() {
        stop_and_exit(app, Some(0)).await;
    }
    Ok(())
}

/// Erases everything the Core stored, then starts the app again. A new launch is the
/// one state in which no window holds data of the erased Core.
#[tauri::command]
pub async fn supervisor_reset<R: Runtime>(
    app: AppHandle<R>,
    supervisor: State<'_, Supervisor>,
    paths: State<'_, Paths>,
    discovery: State<'_, Discovery>,
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
    // The erased Core took the key of the discovery file with it.
    let path = discovery.path.clone();
    blocking(move || discovery::remove(&path)).await?;
    app.restart()
}

/// Returns the API key in the discovery file, or `None` when there is no file.
#[tauri::command]
pub async fn supervisor_api_key(discovery: State<'_, Discovery>) -> Result<Option<String>, String> {
    let path = discovery.path.clone();
    let file = blocking(move || discovery::read(&path)).await?;
    Ok(file.map(|file| file.api_key))
}

/// Writes the discovery file with the address of the running Core and `api_key`, but
/// only while the file holds `replaces`, the key the caller read. `None` means no file.
///
/// Returns false, with no write, when another window changed the key first.
#[tauri::command]
pub async fn supervisor_save_api_key(
    api_key: String,
    replaces: Option<String>,
    supervisor: State<'_, Supervisor>,
    discovery: State<'_, Discovery>,
) -> Result<bool, String> {
    let Status::Running { connection } = supervisor.status() else {
        return Err("Synnax is not running".to_string());
    };
    let file = discovery::File {
        host: connection.host,
        port: connection.port,
        secure: false,
        api_key,
    };
    let path = discovery.path.clone();
    let _guard = discovery.lock.lock().await;
    blocking(move || discovery::swap(&path, &file, replaces.as_deref())).await
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
    fn holds_each_exit_until_the_core_stops_and_stops_it_once() {
        let quit = Quit::default();
        assert_eq!(quit.decide(None), Decision::Stop);
        assert_eq!(quit.decide(Some(0)), Decision::Hold);
        assert!(!quit.start());
    }

    #[test]
    fn lets_the_exit_go_ahead_once_the_core_stopped() {
        let quit = Quit::default();
        quit.decide(None);
        quit.finish();
        assert_eq!(quit.decide(Some(0)), Decision::Exit);
    }

    #[test]
    fn restarts_the_app_for_a_launch_during_the_stop() {
        let quit = Quit::default();
        assert!(quit.start());
        quit.relaunch();
        quit.finish();
        assert_eq!(quit.decide(Some(0)), Decision::Restart);
        assert_eq!(quit.decide(Some(RESTART_EXIT_CODE)), Decision::Exit);
    }

    #[test]
    fn ignores_a_launch_before_the_exit() {
        let quit = Quit::default();
        quit.relaunch();
        quit.decide(None);
        quit.finish();
        assert_eq!(quit.decide(Some(0)), Decision::Exit);
    }

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
