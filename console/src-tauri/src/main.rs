// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(target_os = "macos")]
extern crate objc2;
#[cfg(target_os = "macos")]
extern crate objc2_app_kit;
#[cfg(target_os = "macos")]
extern crate objc2_foundation;

#[cfg(target_os = "macos")]
use device_query::{DeviceEvents, DeviceQuery, DeviceState, MouseState};
#[cfg(target_os = "macos")]
use std::thread;
#[cfg(target_os = "macos")]
use std::time::Duration;
#[cfg(target_os = "macos")]
use tauri::Emitter;

use tauri::Window;

use tauri_plugin_prevent_default::KeyboardShortcut;
use tauri_plugin_prevent_default::ModifierKey::{MetaKey};


#[cfg(target_os = "macos")]
fn set_transparent_titlebar(win: &Window, transparent: bool) {
    use objc2::rc::Retained;
    use objc2::runtime::AnyObject;
    use objc2_app_kit::{
        NSWindow, NSWindowButton, NSWindowStyleMask, NSWindowTitleVisibility,
    };

    let ns_window = win.ns_window().expect("Failed to create window handle") as *mut AnyObject;
    let window: Retained<NSWindow> = unsafe { Retained::retain(ns_window as *mut NSWindow).unwrap() };

    unsafe {
        let mut style_mask = window.styleMask();
        if transparent {
            style_mask.insert(NSWindowStyleMask::FullSizeContentView);
        } else {
            style_mask.remove(NSWindowStyleMask::FullSizeContentView);
        }
        window.setStyleMask(style_mask);

        window.setTitleVisibility(if transparent {
            NSWindowTitleVisibility::Hidden
        } else {
            NSWindowTitleVisibility::Visible
        });

        window.setTitlebarAppearsTransparent(true);

        if let Some(close) = window.standardWindowButton(NSWindowButton::CloseButton) {
            close.removeFromSuperview();
        }
        if let Some(miniaturize) = window.standardWindowButton(NSWindowButton::MiniaturizeButton) {
            miniaturize.removeFromSuperview();
        }
        if let Some(zoom) = window.standardWindowButton(NSWindowButton::ZoomButton) {
            zoom.removeFromSuperview();
        }
    }
}

#[cfg(not(target_os = "macos"))]
fn set_transparent_titlebar(_: &Window, _: bool) {}

fn main() {
    let prevent = tauri_plugin_prevent_default::Builder::new()
        .shortcut(KeyboardShortcut::with_modifiers("W", &[MetaKey]))
        .build();
    tauri::Builder::default()
        .on_page_load(|window, _| {
            set_transparent_titlebar(&window.window(), true);
        })
        .on_window_event(move |win, event| match event {
            tauri::WindowEvent::Focused { .. } => {
                set_transparent_titlebar(win, true);
            }
            tauri::WindowEvent::ThemeChanged { .. } => {
                set_transparent_titlebar(win, true);
            }
            tauri::WindowEvent::Resized(size) => {
                let monitor = win.current_monitor().unwrap().unwrap();
                let screen = monitor.size();
                if size != screen {
                    set_transparent_titlebar(win, true);
                }
            }
            tauri::WindowEvent::Moved(position) => {
                if position.x != 0 && position.y != 0 {
                    set_transparent_titlebar(win, true);
                }
            }
            _ => (),
        })
        .plugin(prevent)
        .plugin(tauri_plugin_single_instance::init(|_app, _argv, _cwd| {}))
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_process::init())
        .setup(|app| {
            #[cfg(desktop)]
            app.handle()
                .plugin(tauri_plugin_updater::Builder::new().build())?;
            #[cfg(target_os = "macos")]
            let app_handle = app.handle().clone();
            #[cfg(target_os = "macos")]
            thread::spawn(move || {
                let app_handle = app_handle.clone();
                let device_state = DeviceState::new();
                let _guard = device_state.on_mouse_up(move |_pos| {
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
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
