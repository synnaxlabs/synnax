// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#![cfg_attr(
  all(not(debug_assertions), target_os = "windows"),
  windows_subsystem = "windows"
)]

#[cfg(target_os = "macos")]
#[macro_use]
extern crate objc;

use tauri::{Runtime, Window};
mod kv;

pub trait WindowExt {
  fn set_transparent_titlebar(&self, transparent: bool);
	fn position_traffic_lights(&self, x: f64, y: f64);
}

impl<R: Runtime> WindowExt for Window<R> {
	#[cfg(target_os = "linux")]
	fn set_transparent_titlebar(&self, _transparent: bool) {}

	#[cfg(target_os = "windows")]
	fn set_transparent_titlebar(&self, _transparent: bool) {}

  #[cfg(target_os = "macos")]
  fn set_transparent_titlebar(&self, transparent: bool) {
		use cocoa::appkit::{NSWindow, NSWindowStyleMask, NSWindowTitleVisibility};

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
    }
  }

	#[cfg(target_os = "linux")]
	fn position_traffic_lights(&self, _x: f64, _y: f64) {}

	#[cfg(target_os = "windows")]
	fn position_traffic_lights(&self, _x: f64, _y: f64) {}


	 #[cfg(target_os = "macos")]
	fn position_traffic_lights(&self, x: f64, y: f64) {
    	use cocoa::appkit::{NSView, NSWindow, NSWindowButton};
    	use cocoa::foundation::NSRect;

    	let window = self.ns_window().unwrap() as cocoa::base::id;

    	unsafe {
        	let close = window.standardWindowButton_(NSWindowButton::NSWindowCloseButton);
        	let miniaturize =
            	window.standardWindowButton_(NSWindowButton::NSWindowMiniaturizeButton);
        	let zoom = window.standardWindowButton_(NSWindowButton::NSWindowZoomButton);

        	let title_bar_container_view = close.superview().superview();

        	let close_rect: NSRect = msg_send![close, frame];
        	let button_height = close_rect.size.height;

        	let title_bar_frame_height = button_height + y;
        	let mut title_bar_rect = NSView::frame(title_bar_container_view);
        	title_bar_rect.size.height = title_bar_frame_height;
        	title_bar_rect.origin.y = NSView::frame(window).size.height - title_bar_frame_height;
        	let _: () = msg_send![title_bar_container_view, setFrame: title_bar_rect];

        	let window_buttons = vec![close, miniaturize, zoom];
        	let space_between = NSView::frame(miniaturize).origin.x - NSView::frame(close).origin.x;

        	for (i, button) in window_buttons.into_iter().enumerate() {
            	let mut rect: NSRect = NSView::frame(button);
            	rect.origin.x = x + (i as f64 * space_between);
            	button.setFrameOrigin(rect.origin);
        	}
    	}
	}

}



fn main() {
    let db = kv::open().unwrap();
    tauri::Builder::default()
        .manage(db)
        .invoke_handler(tauri::generate_handler![kv::kv_exec])
    .on_window_event(|event| match event.event() {
         tauri::WindowEvent::Focused {..} => {
            event.window().set_transparent_titlebar(true);
						event.window().position_traffic_lights(15.0, 17.5);
         }
         _ => {}
      })
     .run(tauri::generate_context!())
      .expect("error while running tauri application");
}
