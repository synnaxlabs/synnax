// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(target_os = "macos")]
extern crate cocoa;

#[cfg(target_os = "macos")]
use device_query::{DeviceEvents, DeviceQuery, DeviceState, MouseState};
#[cfg(target_os = "macos")]
use std::thread;
#[cfg(target_os = "macos")]
use std::time::Duration;
#[cfg(target_os = "macos")]
use tauri::Emitter;
#[cfg(target_os = "macos")]
struct UnsafeWindowHandle(*mut std::ffi::c_void);
#[cfg(target_os = "macos")]
unsafe impl Send for UnsafeWindowHandle {}
#[cfg(target_os = "macos")]
unsafe impl Sync for UnsafeWindowHandle {}

use tauri::Window;

#[cfg(target_os = "macos")]
fn set_transparent_titlebar(win: &Window, transparent: bool) {
    let ns_window_handle =
        UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle"));
    use cocoa::appkit::{
        NSView, NSWindow, NSWindowButton, NSWindowStyleMask, NSWindowTitleVisibility,
    };
    let id = ns_window_handle.0 as cocoa::base::id;
    unsafe {
        let mut style_mask = id.styleMask();
        style_mask.set(
            NSWindowStyleMask::NSFullSizeContentViewWindowMask,
            transparent,
        );
        id.setStyleMask_(style_mask);
        id.setTitleVisibility_(if transparent {
            NSWindowTitleVisibility::NSWindowTitleHidden
        } else {
            NSWindowTitleVisibility::NSWindowTitleVisible
        });
        id.setTitlebarAppearsTransparent_(cocoa::base::YES);
        let close = id.standardWindowButton_(NSWindowButton::NSWindowCloseButton);
        let miniaturize = id.standardWindowButton_(NSWindowButton::NSWindowMiniaturizeButton);
        let zoom = id.standardWindowButton_(NSWindowButton::NSWindowZoomButton);
        let window_buttons = vec![close, miniaturize, zoom];
        for button in window_buttons {
            button.removeFromSuperview()
        }
    }
}

#[cfg(not(target_os = "macos"))]
fn set_transparent_titlebar(_: &Window, _: bool) {}

fn main() {
    tauri::Builder::default()
        .on_page_load(|window, _| {
            set_transparent_titlebar(&window.window(), true);
            return;
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
