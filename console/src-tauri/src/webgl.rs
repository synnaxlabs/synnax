// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Turns on WebGL in workers for every WebKitGTK webview. Pluto renders from a worker,
//! and WebKitGTK turns WebGL in workers off by default.

use std::ffi::{CStr, c_char, c_int};

use tauri::{Runtime, plugin::TauriPlugin};
use webkit2gtk::{WebViewExt, ffi::WebKitSettings, glib::translate::ToGlibPtr};

const FEATURE: &CStr = c"AllowWebGLInWorkers";

#[repr(C)]
struct FeatureList {
    _private: [u8; 0],
}

#[repr(C)]
struct Feature {
    _private: [u8; 0],
}

// WebKitGTK 2.42 added the feature API, which the webkit2gtk crate does not bind.
unsafe extern "C" {
    fn webkit_settings_get_all_features() -> *mut FeatureList;
    fn webkit_feature_list_get_length(list: *mut FeatureList) -> usize;
    fn webkit_feature_list_get(list: *mut FeatureList, index: usize) -> *mut Feature;
    fn webkit_feature_list_unref(list: *mut FeatureList);
    fn webkit_feature_get_identifier(feature: *mut Feature) -> *const c_char;
    fn webkit_settings_set_feature_enabled(
        settings: *mut WebKitSettings,
        feature: *mut Feature,
        enabled: c_int,
    );
}

fn allow_in_workers(webview: &webkit2gtk::WebView) {
    let settings = webview
        .settings()
        .expect("WebKitGTK webview has no settings");
    let mut found = false;
    // SAFETY: the list owns its features until the unref, and settings outlives the
    // call.
    unsafe {
        let list = webkit_settings_get_all_features();
        for i in 0..webkit_feature_list_get_length(list) {
            let feature = webkit_feature_list_get(list, i);
            if CStr::from_ptr(webkit_feature_get_identifier(feature)) == FEATURE {
                webkit_settings_set_feature_enabled(settings.to_glib_none().0, feature, 1);
                found = true;
            }
        }
        webkit_feature_list_unref(list);
    }
    assert!(found, "WebKitGTK has no {FEATURE:?} feature");
}

/// Builds a plugin that turns on WebGL in workers for each webview before its page
/// runs scripts.
pub fn plugin<R: Runtime>() -> TauriPlugin<R> {
    tauri::plugin::Builder::new("webgl")
        .on_webview_ready(|webview| {
            webview
                .with_webview(|platform| allow_in_workers(&platform.inner()))
                .expect("failed to reach the WebKitGTK webview");
        })
        .build()
}
