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
#[cfg(target_os = "macos")]
mod mouse;
#[cfg(feature = "desktop")]
mod supervisor;
#[cfg(not(feature = "desktop"))]
mod updater;

use tauri::Manager;

use tauri_plugin_prevent_default::KeyboardShortcut;
use tauri_plugin_prevent_default::ModifierKey::MetaKey;

fn main() {
    // WebKitGTK's DMABUF renderer draws a black window when the WebKitGTK bundled in an
    // AppImage runs on the host's graphics stack.
    #[cfg(target_os = "linux")]
    if std::env::var_os("APPIMAGE").is_some() {
        // SAFETY: no other thread runs yet.
        unsafe { std::env::set_var("WEBKIT_DISABLE_DMABUF_RENDERER", "1") };
    }
    let prevent = tauri_plugin_prevent_default::Builder::new()
        .shortcut(KeyboardShortcut::with_modifiers("W", &[MetaKey]))
        .build();
    let builder = tauri::Builder::default();
    #[cfg(not(feature = "desktop"))]
    let builder = builder.invoke_handler(tauri::generate_handler![
        keychain::keychain_get,
        keychain::keychain_set,
        keychain::keychain_delete,
        #[cfg(target_os = "macos")]
        mouse::mouse_release,
        updater::updater_check,
    ]);
    #[cfg(feature = "desktop")]
    let builder = builder.invoke_handler(tauri::generate_handler![
        keychain::keychain_get,
        keychain::keychain_set,
        keychain::keychain_delete,
        #[cfg(target_os = "macos")]
        mouse::mouse_release,
        supervisor::commands::supervisor_status,
        supervisor::commands::supervisor_history,
        supervisor::commands::supervisor_restart,
        supervisor::commands::supervisor_stop,
        supervisor::commands::supervisor_quit,
        supervisor::commands::supervisor_reset,
        supervisor::commands::supervisor_show_logs,
        supervisor::commands::supervisor_show_data,
        supervisor::commands::supervisor_diagnostics,
        supervisor::commands::supervisor_log_tail,
        supervisor::commands::supervisor_export_diagnostics,
        supervisor::commands::supervisor_api_key,
        supervisor::commands::supervisor_save_api_key,
        supervisor::commands::supervisor_forget_legacy_api_key,
        supervisor::commands::supervisor_legacy_api_key,
        install::install_info,
    ]);
    #[cfg(target_os = "linux")]
    let builder = builder.plugin(desktop::webkit::init());
    builder
        .plugin(prevent)
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            #[cfg(feature = "desktop")]
            supervisor::commands::relaunched(app);
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
            // The .deb registers the app's scheme in its desktop file, but an AppImage
            // installs none, so it registers the scheme itself. A host without the
            // desktop tools only loses links to the app.
            #[cfg(target_os = "linux")]
            if std::env::var_os("APPIMAGE").is_some() {
                use tauri_plugin_deep_link::DeepLinkExt;
                if let Err(e) = app.deep_link().register_all() {
                    let schemes = app
                        .config()
                        .plugins
                        .0
                        .get("deep-link")
                        .and_then(|c| c.pointer("/desktop/schemes"))
                        .and_then(|s| s.as_array())
                        .into_iter()
                        .flatten()
                        .filter_map(|s| s.as_str())
                        .map(|s| format!("{s}://"))
                        .collect::<Vec<_>>()
                        .join(", ");
                    eprintln!(
                        "failed to register {schemes}: {e}. Install desktop-file-utils \
                         and xdg-utils to open {schemes} links."
                    );
                }
            }
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app, _event| {
            #[cfg(feature = "desktop")]
            match _event {
                tauri::RunEvent::ExitRequested { code, api, .. } => {
                    supervisor::commands::exit_requested(_app, code, &api);
                }
                // Windows ends a session with `Exit` and no `ExitRequested` first.
                tauri::RunEvent::Exit => supervisor::commands::shutdown(_app),
                // macOS opens a running app again with `Reopen`, not a second process.
                #[cfg(target_os = "macos")]
                tauri::RunEvent::Reopen { .. } => supervisor::commands::relaunched(_app),
                _ => {}
            }
        });
}
