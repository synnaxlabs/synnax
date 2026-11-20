// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Pluto draws with WebGL on an OffscreenCanvas in a worker, which WebKitGTK allows
//! only when its AllowWebGLInWorkers feature is on. The feature is off by default.

use std::ffi::{CStr, c_char};

use tauri::Runtime;
use tauri::plugin::{Builder, TauriPlugin};
use webkit2gtk::glib::translate::ToGlibPtr;
use webkit2gtk::{Settings, WebViewExt};

const FEATURE: &CStr = c"AllowWebGLInWorkers";

#[repr(C)]
struct FeatureList([u8; 0]);

#[repr(C)]
struct Feature([u8; 0]);

// WebKitGTK 2.42 added the feature API, which the webkit2gtk crate does not bind.
unsafe extern "C" {
    fn webkit_settings_get_all_features() -> *mut FeatureList;
    fn webkit_feature_list_get_length(list: *mut FeatureList) -> usize;
    fn webkit_feature_list_get(list: *mut FeatureList, index: usize) -> *mut Feature;
    fn webkit_feature_list_unref(list: *mut FeatureList);
    fn webkit_feature_get_identifier(feature: *mut Feature) -> *const c_char;
    fn webkit_settings_set_feature_enabled(
        settings: *mut webkit2gtk::ffi::WebKitSettings,
        feature: *mut Feature,
        enabled: webkit2gtk::glib::ffi::gboolean,
    );
}

fn enable(settings: &Settings) {
    // SAFETY: the list owns each feature, and it lives until the unref at the end.
    unsafe {
        let list = webkit_settings_get_all_features();
        let feature = (0..webkit_feature_list_get_length(list))
            .map(|i| webkit_feature_list_get(list, i))
            .find(|&f| CStr::from_ptr(webkit_feature_get_identifier(f)) == FEATURE);
        match feature {
            Some(f) => webkit_settings_set_feature_enabled(
                settings.to_glib_none().0,
                f,
                webkit2gtk::glib::ffi::GTRUE,
            ),
            None => eprintln!("WebKitGTK has no {FEATURE:?} feature"),
        }
        webkit_feature_list_unref(list);
    }
}

/// Turns on WebGL in workers for each webview the app creates.
pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("webkit")
        .on_webview_ready(|webview| {
            let result = webview.with_webview(|platform| {
                if let Some(settings) = platform.inner().settings() {
                    enable(&settings);
                }
            });
            if let Err(e) = result {
                eprintln!("failed to turn on WebGL in workers: {e}");
            }
        })
        .build()
}
