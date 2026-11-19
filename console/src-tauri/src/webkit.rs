// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Configures WebKitGTK for Pluto, which draws with WebGL in a worker and hands each
//! frame to the page as bitmaps. Without a GPU render node, the worker gets no WebGL,
//! and Pluto shows that a GPU is required.

use std::ffi::{CStr, c_char};
use std::path::Path;

use tauri::Runtime;
use tauri::plugin::{Builder, TauriPlugin};
use webkit2gtk::glib::translate::ToGlibPtr;
use webkit2gtk::{Settings, WebViewExt};

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

/// Sets the WebKitGTK environment. The WebKitGTK bundled in an AppImage draws a black
/// window with its DMABUF renderer on.
///
/// # Safety
///
/// Call it before any other thread starts.
pub unsafe fn set_env() {
    if std::env::var_os("APPIMAGE").is_some() {
        // SAFETY: the caller guarantees that no other thread runs.
        unsafe { std::env::set_var("WEBKIT_DISABLE_DMABUF_RENDERER", "1") };
    }
}

fn has_render_node(dri: &Path) -> bool {
    dri.read_dir().is_ok_and(|entries| {
        entries
            .flatten()
            .any(|e| e.file_name().as_encoded_bytes().starts_with(b"renderD"))
    })
}

/// The features to set on each webview, for a WebKitGTK of the given major and minor
/// version.
fn features(render_node: bool, version: (u32, u32)) -> Vec<(&'static CStr, bool)> {
    if !render_node {
        return Vec::new();
    }
    let mut features = vec![
        (c"AllowWebGLInWorkers", true),
        // Accelerated 2D canvases crash the web process during the bitmap handoff.
        (c"CanvasUsesAcceleratedDrawing", false),
    ];
    // WebKitGTK 2.54 runs worker WebGL only in the GPU process, which crashes 2.50.
    // Feature identifiers drop the "Enabled" suffix of WebKit's preference keys.
    if version >= (2, 54) {
        features.push((c"UseGPUProcessForWebGL", true));
    }
    features
}

fn set(settings: &Settings, features: &[(&CStr, bool)]) {
    // SAFETY: the list owns each feature, and it lives until the unref at the end.
    unsafe {
        let list = webkit_settings_get_all_features();
        let all: Vec<_> = (0..webkit_feature_list_get_length(list))
            .map(|i| webkit_feature_list_get(list, i))
            .collect();
        for &(id, enabled) in features {
            match all
                .iter()
                .find(|&&f| CStr::from_ptr(webkit_feature_get_identifier(f)) == id)
            {
                Some(&f) => webkit_settings_set_feature_enabled(
                    settings.to_glib_none().0,
                    f,
                    enabled.into(),
                ),
                None => panic!("WebKitGTK has no {id:?} feature"),
            }
        }
        webkit_feature_list_unref(list);
    }
}

/// Sets the WebKitGTK features that Pluto needs on each webview the app creates. The
/// app panics when it creates a webview if WebKitGTK lacks one of the features.
pub fn init<R: Runtime>() -> TauriPlugin<R> {
    // SAFETY: the functions only read constants.
    let version = unsafe {
        (
            webkit2gtk::ffi::webkit_get_major_version(),
            webkit2gtk::ffi::webkit_get_minor_version(),
        )
    };
    let features = features(has_render_node(Path::new("/dev/dri")), version);
    Builder::new("webkit")
        .on_webview_ready(move |webview| {
            let result = webview.with_webview({
                let features = features.clone();
                move |platform| {
                    if let Some(settings) = platform.inner().settings() {
                        set(&settings, &features);
                    }
                }
            });
            if let Err(e) = result {
                eprintln!("failed to set the WebKitGTK features: {e}");
            }
        })
        .build()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn finds_a_render_node() {
        let dri = tempfile::tempdir().unwrap();
        std::fs::write(dri.path().join("card0"), "").unwrap();
        assert!(!has_render_node(dri.path()));
        std::fs::write(dri.path().join("renderD128"), "").unwrap();
        assert!(has_render_node(dri.path()));
    }

    #[test]
    fn finds_no_render_node_without_a_dri_directory() {
        let dri = tempfile::tempdir().unwrap();
        assert!(!has_render_node(&dri.path().join("dri")));
    }

    #[test]
    fn leaves_worker_webgl_off_without_a_render_node() {
        assert!(features(false, (2, 54)).is_empty());
    }

    #[test]
    fn turns_on_worker_webgl_without_the_gpu_process_before_2_54() {
        assert_eq!(
            features(true, (2, 52)),
            [
                (c"AllowWebGLInWorkers", true),
                (c"CanvasUsesAcceleratedDrawing", false),
            ]
        );
    }

    #[test]
    fn runs_worker_webgl_in_the_gpu_process_from_2_54() {
        assert!(features(true, (2, 54)).contains(&(c"UseGPUProcessForWebGL", true)));
    }
}
