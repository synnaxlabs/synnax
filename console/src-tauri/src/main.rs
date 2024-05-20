// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(target_os = "macos")]
#[macro_use]
extern crate cocoa;

#[cfg(target_os = "macos")]
#[macro_use]
extern crate objc;

use objc::{msg_send, sel, sel_impl};
use rand::{distributions::Alphanumeric, Rng};
use tauri::{
    plugin::{Builder, TauriPlugin},
    Manager, Runtime, Window,
}; // 0.8

struct UnsafeWindowHandle(*mut std::ffi::c_void);
unsafe impl Send for UnsafeWindowHandle {}
unsafe impl Sync for UnsafeWindowHandle {}

#[cfg(target_os = "macos")]
fn position_traffic_lights(win: Window, transparent: bool) {
    let ns_window_handle = UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle"));
    use cocoa::appkit::{NSView, NSWindow, NSWindowButton, NSWindowStyleMask, NSWindowTitleVisibility};
    use cocoa::foundation::NSRect;
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
        let miniaturize =
            id.standardWindowButton_(NSWindowButton::NSWindowMiniaturizeButton);
        let zoom = id.standardWindowButton_(NSWindowButton::NSWindowZoomButton);
        let window_buttons = vec![close, miniaturize, zoom];
        for button in window_buttons {
            button.removeFromSuperview()
        }
    }
}

fn main() {
    tauri::Builder::default()
        .on_page_load(|window, _| {
            #[cfg(target_os = "macos")]
            position_traffic_lights(
                UnsafeWindowHandle(window.window().ns_window().expect("Failed to create window handle")),
                true,
            );
            return;
        })
        .on_window_event(move |win, event| match event {
            tauri::WindowEvent::Focused {..} => {
                position_traffic_lights(win, true);
            },
            tauri::WindowEvent::ThemeChanged {..} => {
                position_traffic_lights(win, true);
            }
            tauri::WindowEvent::Resized(size) => {
                let monitor = win.current_monitor().unwrap().unwrap();
                let screen = monitor.size();
                if size != screen {
                    position_traffic_lights(win, true);
                } 
            },
            tauri::WindowEvent::Moved(position)=> {
                if position.x != 0 && position.y != 0 {
                    position_traffic_lights(win, true);
                }
           },
            _ => (),
         })
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
