// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Where a drag released outside every window ended, which macOS does not tell the
//! webview.

use std::thread;
use std::time::Duration;

use objc2_core_graphics::{CGEvent, CGEventSource, CGEventSourceStateID, CGMouseButton};

const POLL_INTERVAL: Duration = Duration::from_millis(10);

/// Resolves with the cursor's screen position once the left mouse button is up, in
/// points from the top left of the main display.
#[tauri::command]
pub async fn mouse_release() -> Result<(f64, f64), String> {
    // Polling needs no permission. An event tap would need Accessibility.
    tauri::async_runtime::spawn_blocking(|| {
        while CGEventSource::button_state(
            CGEventSourceStateID::CombinedSessionState,
            CGMouseButton::Left,
        ) {
            thread::sleep(POLL_INTERVAL);
        }
        let event = CGEvent::new(None).ok_or("failed to read the cursor position")?;
        let point = CGEvent::location(Some(&event));
        Ok((point.x, point.y))
    })
    .await
    .map_err(|e| e.to_string())?
}
