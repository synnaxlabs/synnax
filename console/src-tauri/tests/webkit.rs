// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Draws frames in a worker and hands them to the page as bitmaps, as Pluto does, in a
//! webview with the Console's WebKitGTK plugin. With a GPU render node, fails if the
//! web process crashes or the page stops drawing, or if Pluto's line shaders compute
//! wrong positions for nanosecond timestamps. Without one, fails unless the workers get
//! no WebGL.

#[cfg(not(target_os = "linux"))]
fn main() {}

#[cfg(target_os = "linux")]
fn main() {
    use std::time::Duration;

    use tauri::http::Response;
    use tauri::{WebviewUrl, WebviewWindowBuilder};
    use webkit2gtk::WebViewExt;

    const SECONDS: u64 = 60;
    // A page that draws fewer frames did not run long enough to show a crash.
    const MIN_FRAMES: u64 = 10 * SECONDS;

    let render_node = std::fs::read_dir("/dev/dri").is_ok_and(|entries| {
        entries
            .flatten()
            .any(|e| e.file_name().as_encoded_bytes().starts_with(b"renderD"))
    });
    let mut context = tauri::generate_context!();
    context.config_mut().app.windows.clear();
    let app = tauri::Builder::default()
        .plugin(desktop::webkit::init())
        .register_uri_scheme_protocol("canvas", |_, request| {
            let (body, kind) = match request.uri().path() {
                "/worker.js" => (include_str!("webkit/worker.js"), "text/javascript"),
                "/precision.html" => {
                    (include_str!("webkit/precision.html"), "text/html")
                }
                "/precision.js" => {
                    (include_str!("webkit/precision.js"), "text/javascript")
                }
                "/vert_f32.glsl" => (
                    include_str!("../../../pluto/src/vis/line/aether/vert_f32.glsl"),
                    "text/plain",
                ),
                "/vert_hybrid.glsl" => (
                    include_str!("../../../pluto/src/vis/line/aether/vert_hybrid.glsl"),
                    "text/plain",
                ),
                "/frag.glsl" => (
                    include_str!("../../../pluto/src/vis/line/aether/frag.glsl"),
                    "text/plain",
                ),
                _ => (include_str!("webkit/canvas.html"), "text/html"),
            };
            Response::builder()
                .header("Content-Type", kind)
                .body(body.as_bytes().to_vec())
                .unwrap()
        })
        .build(context)
        .expect("failed to build the app");
    let url = WebviewUrl::CustomProtocol("canvas://localhost/".parse().unwrap());
    let window = WebviewWindowBuilder::new(&app, "main", url)
        .inner_size(1600.0, 1000.0)
        .build()
        .expect("failed to open the window");
    let handle = app.handle().clone();
    window
        .with_webview(move |webview| {
            webview
                .inner()
                .connect_web_process_terminated(move |_, reason| {
                    eprintln!("web process terminated: {reason:?}");
                    handle.exit(1);
                });
        })
        .unwrap();
    let url = WebviewUrl::CustomProtocol(
        "canvas://localhost/precision.html".parse().unwrap(),
    );
    let precision = WebviewWindowBuilder::new(&app, "precision", url)
        .build()
        .expect("failed to open the precision window");
    let handle = app.handle().clone();
    std::thread::spawn(move || {
        std::thread::sleep(Duration::from_secs(SECONDS));
        let title = |window: &tauri::WebviewWindow| {
            let (tx, rx) = std::sync::mpsc::channel();
            window
                .with_webview(move |webview| {
                    let title = webview.inner().title().unwrap_or_default();
                    tx.send(title.to_string()).unwrap();
                })
                .unwrap();
            rx.recv().unwrap()
        };
        let (frames, precision) = (title(&window), title(&precision));
        eprintln!("render node: {render_node}, {frames} in {SECONDS} s, {precision}");
        let passed = if render_node {
            let drew = frames
                .strip_prefix("frames ")
                .and_then(|n| n.parse::<u64>().ok())
                .is_some_and(|frames| frames >= MIN_FRAMES);
            drew && precision == "precision ok"
        } else {
            frames == "no webgl" && precision == "no webgl"
        };
        handle.exit(if passed { 0 } else { 1 });
    });
    std::process::exit(app.run_return(|_, _| {}));
}
