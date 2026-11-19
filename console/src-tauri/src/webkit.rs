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
use webkit2gtk::glib::subclass::SignalId;
use webkit2gtk::glib::translate::{FromGlibPtrBorrow, IntoGlib, ToGlibPtr};
use webkit2gtk::glib::{self, Cast, Class, StaticType, gobject_ffi};
use webkit2gtk::{Settings, WebView, WebViewExt};

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
    // SAFETY: the caller guarantees that no other thread runs.
    unsafe {
        // A GPU painting thread frees its GL context when the web process exits, at
        // the same time as the NVIDIA driver unloads, and crashes. The main thread
        // paints instead and frees its context before the driver unloads.
        std::env::set_var("WEBKIT_SKIA_GPU_PAINTING_THREADS", "0");
        if std::env::var_os("APPIMAGE").is_some() {
            std::env::set_var("WEBKIT_DISABLE_DMABUF_RENDERER", "1");
        }
    }
}

fn has_render_node(dri: &Path) -> bool {
    dri.read_dir().is_ok_and(|entries| {
        entries
            .flatten()
            .any(|e| e.file_name().as_encoded_bytes().starts_with(b"renderD"))
    })
}

type Features = Vec<(&'static CStr, bool)>;

/// The features to set on each webview, for a WebKitGTK of the given major and minor
/// version.
fn features(render_node: bool, version: (u32, u32)) -> Features {
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

unsafe extern "C" fn on_parent_set(
    _hint: *mut gobject_ffi::GSignalInvocationHint,
    _n_params: u32,
    params: *const gobject_ffi::GValue,
    features: glib::ffi::gpointer,
) -> glib::ffi::gboolean {
    // SAFETY: the first parameter holds the widget that emits the signal, and the hook
    // owns the features.
    let (widget, features) = unsafe {
        (
            glib::Object::from_glib_borrow(gobject_ffi::g_value_get_object(params)),
            &*features.cast::<Features>(),
        )
    };
    let view = widget.downcast_ref::<WebView>();
    if let Some(settings) = view.and_then(|v| v.settings()) {
        set(&settings, features);
    }
    glib::ffi::GTRUE
}

unsafe extern "C" fn drop_features(features: glib::ffi::gpointer) {
    // SAFETY: the pointer comes from the Box::into_raw in init.
    drop(unsafe { Box::from_raw(features.cast::<Features>()) });
}

/// Sets the WebKitGTK features that Pluto needs on each webview the app creates. The
/// app panics when it creates a webview if WebKitGTK lacks one of the features.
pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("webkit")
        .setup(|_, _| {
            // SAFETY: the functions only read constants.
            let version = unsafe {
                (
                    webkit2gtk::ffi::webkit_get_major_version(),
                    webkit2gtk::ffi::webkit_get_minor_version(),
                )
            };
            let features = features(has_render_node(Path::new("/dev/dri")), version);
            if features.is_empty() {
                return Ok(());
            }
            // WebKitGTK reads the features only when a webview starts its web process,
            // on its first page load. wry adds each webview to its window just before
            // that load, so a hook on parent-set reaches every webview in time.
            let class = Class::<WebView>::from_type(WebView::static_type())
                .expect("failed to load the WebKitGTK webview class");
            let signal = SignalId::lookup("parent-set", class.type_())
                .expect("GTK has no parent-set signal");
            // SAFETY: GTK emits the signal on this thread, and drop_features frees the
            // features only when the hook goes away.
            unsafe {
                gobject_ffi::g_signal_add_emission_hook(
                    signal.into_glib(),
                    0,
                    Some(on_parent_set),
                    Box::into_raw(Box::new(features)).cast(),
                    Some(drop_features),
                );
            }
            Ok(())
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
