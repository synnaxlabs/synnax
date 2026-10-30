// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(feature = "desktop")]
mod install;
mod keychain;
#[cfg(feature = "desktop")]
mod supervisor;

#[cfg(target_os = "macos")]
use device_query::{DeviceEvents, DeviceEventsHandler, DeviceQuery, DeviceState, MouseState};
#[cfg(target_os = "macos")]
use std::thread;
#[cfg(target_os = "macos")]
use std::time::Duration;
#[cfg(target_os = "macos")]
use tauri::Emitter;

use tauri::Manager;

use tauri_plugin_prevent_default::KeyboardShortcut;
use tauri_plugin_prevent_default::ModifierKey::MetaKey;

fn main() {
    let prevent = tauri_plugin_prevent_default::Builder::new()
        .shortcut(KeyboardShortcut::with_modifiers("W", &[MetaKey]))
        .build();
    let builder = tauri::Builder::default();
    #[cfg(not(feature = "desktop"))]
    let builder = builder.invoke_handler(tauri::generate_handler![
        keychain::keychain_get,
        keychain::keychain_set,
        keychain::keychain_delete,
    ]);
    #[cfg(feature = "desktop")]
    let builder = builder.invoke_handler(tauri::generate_handler![
        keychain::keychain_get,
        keychain::keychain_set,
        keychain::keychain_delete,
        supervisor::commands::supervisor_status,
        supervisor::commands::supervisor_history,
        supervisor::commands::supervisor_restart,
        supervisor::commands::supervisor_stop,
        supervisor::commands::supervisor_reset,
        supervisor::commands::supervisor_show_logs,
        supervisor::commands::supervisor_show_data,
        supervisor::commands::supervisor_diagnostics,
        supervisor::commands::supervisor_log_tail,
        supervisor::commands::supervisor_export_diagnostics,
        install::install_info,
    ]);
    builder
        .plugin(prevent)
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.unminimize();
                let _ = win.show();
                let _ = win.set_focus();
            }
        }))
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_process::init())
        .setup(|app| {
            #[cfg(feature = "desktop")]
            install::init(app.handle());
            #[cfg(feature = "desktop")]
            supervisor::commands::init(app.handle())?;
            #[cfg(desktop)]
            app.handle()
                .plugin(tauri_plugin_updater::Builder::new().build())?;
            #[cfg(target_os = "macos")]
            let app_handle = app.handle().clone();
            #[cfg(target_os = "macos")]
            thread::spawn(move || {
                let app_handle = app_handle.clone();
                let handler = DeviceEventsHandler::new(Duration::from_millis(10))
                    .expect("Failed to create device events handler");
                let _guard = handler.on_mouse_up(move |_button| {
                    let state: MouseState = DeviceState::new().get_mouse();
                    app_handle
                        .emit("mouse_up", state.coords)
                        .expect("Failed to emit event");
                });
                loop {
                    thread::sleep(Duration::from_secs(1));
                }
            });
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app, _event| {
            #[cfg(feature = "desktop")]
            if let tauri::RunEvent::Exit = _event {
                supervisor::commands::shutdown(_app);
            }
        });
}
