// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

use serde::Serialize;
use tauri::{Manager, ResourceId, Runtime, Url, Webview};
use tauri_plugin_updater::{Error, UpdaterExt};

/// The update in the shape that the updater plugin's JavaScript `Update` class takes.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Metadata {
    rid: ResourceId,
    current_version: String,
    version: String,
    raw_json: serde_json::Value,
}

/// Checks the manifest at endpoint for a release whose version differs from the running
/// Console, older or newer. Returns None when the endpoint serves no manifest, or none
/// for this platform. The plugin's download and install commands take the update by its
/// rid.
#[tauri::command]
pub async fn updater_check<R: Runtime>(
    webview: Webview<R>,
    endpoint: Url,
) -> Result<Option<Metadata>, Error> {
    let updater = webview
        .updater_builder()
        .endpoints(vec![endpoint])?
        .version_comparator(|current, release| release.version != current)
        .build()?;
    let update = match updater.check().await {
        Ok(update) => update,
        // A Core answers 404 when it embeds no Console update. The plugin reports every
        // non-2xx answer as ReleaseNotFound, and a manifest with no build for this
        // platform as TargetNotFound.
        Err(Error::ReleaseNotFound | Error::TargetNotFound(_) | Error::TargetsNotFound(_)) => None,
        Err(err) => return Err(err),
    };
    Ok(update.map(|update| Metadata {
        current_version: update.current_version.clone(),
        version: update.version.clone(),
        raw_json: update.raw_json.clone(),
        rid: webview.resources_table().add(update),
    }))
}
