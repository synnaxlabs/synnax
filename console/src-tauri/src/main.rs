// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(target_os = "macos")]
extern crate cocoa;

#[cfg(target_os = "macos")]
extern crate objc;

struct UnsafeWindowHandle(*mut std::ffi::c_void);
unsafe impl Send for UnsafeWindowHandle {}
unsafe impl Sync for UnsafeWindowHandle {}

#[cfg(target_os = "macos")]
fn position_traffic_lights(ns_window_handle: UnsafeWindowHandle, transparent: bool) {
    use cocoa::appkit::{NSView, NSWindow, NSWindowButton, NSWindowStyleMask, NSWindowTitleVisibility};
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



    



// Learn more about Tauri commands at https://tauri.app/v1/guides/features/command
#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! You've been greeted from Rust!", name)
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
                position_traffic_lights(
                    UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle")),
                    true,
                );
            },
            tauri::WindowEvent::ThemeChanged {..} => {
                position_traffic_lights(
                    UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle")),
                    true,
                );
            }
            tauri::WindowEvent::Resized(size) => {
                let monitor = win.current_monitor().unwrap().unwrap();
                let screen = monitor.size();
                if size != screen {
                    position_traffic_lights(
                        UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle")),
                        true,
                    );
                } 

                
                
            },
            tauri::WindowEvent::Moved(position)=> {
                if position.x != 0 && position.y != 0 {
                    position_traffic_lights(
                        UnsafeWindowHandle(win.ns_window().expect("Failed to create window handle")),
                        true,
                    );
                }
           },
            _ => (),
         })
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .invoke_handler(tauri::generate_handler![greet])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
