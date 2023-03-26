/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

#![cfg_attr(
  all(not(debug_assertions), target_os = "windows"),
  windows_subsystem = "windows"
)]

#[cfg(target_os = "macos")]
extern crate objc;

use tauri::{Runtime, Window};
mod kv;

pub trait WindowExt {
  fn set_transparent_titlebar(&self, transparent: bool);
}

impl<R: Runtime> WindowExt for Window<R> {
	#[cfg(target_os = "linux")]
	fn set_transparent_titlebar(&self, _transparent: bool) {}

	#[cfg(target_os = "windows")]
	fn set_transparent_titlebar(&self, _transparent: bool) {}

    #[cfg(target_os = "macos")]
    fn set_transparent_titlebar(&self, transparent: bool) {
        use cocoa::appkit::{NSWindow, NSWindowStyleMask, NSWindowTitleVisibility, NSWindowButton, NSView};

        unsafe {
            let id = self.ns_window().unwrap() as cocoa::base::id;

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
            id.setTitlebarAppearsTransparent_(if transparent {
                cocoa::base::YES
            } else {
                cocoa::base::NO
            });

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
}

#[derive(Clone, serde::Serialize)]
struct Payload {
    message: String,
}

fn main() {
    let mut builder = tauri::Builder::default();
    let mut db_err: String = "".to_string();
    match kv::open() {
        Ok(db) => builder = builder.manage(db).invoke_handler(tauri::generate_handler![kv::kv_exec]),
        Err(e) => db_err = e,
    };
    builder
    .on_window_event(move |event| match event.event() {
         tauri::WindowEvent::Focused {..} => {
            event.window().set_transparent_titlebar(true);
         },
        tauri::WindowEvent::ThemeChanged {..} => {
            event.window().set_transparent_titlebar(true);
         }
         tauri::WindowEvent::Resized(size) => {
            let monitor = event.window().current_monitor().unwrap().unwrap();
            let screen = monitor.size();
            if size != screen {
                event.window().set_transparent_titlebar(true);
            } 
         },
         tauri::WindowEvent::Moved(position)=> {
            if position.x != 0 || position.y != 0 {
                event.window().set_transparent_titlebar(true);
            }
        },
         _ => (),
      })
    .on_page_load(move |window, _| {
        window.set_transparent_titlebar(true);
        if window.label() != "main" { return };
        let db_err_ = db_err.clone();
        let win = window.clone();
        window.listen("kv_open_req", move |_| {
            let db_err__ = db_err_.clone(); 
            win.emit("kv_open_res", Some(Payload { message: db_err__ })).unwrap();
        });
    })
     .run(tauri::generate_context!())
      .expect("error while running tauri application");
}
