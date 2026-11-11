// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

use semver::Version;
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
    /// True when the release is older than the running Console.
    downgrade: bool,
    raw_json: serde_json::Value,
}

fn minor(version: &Version) -> (u64, u64) {
    (version.major, version.minor)
}

/// Reports whether a Console at current may install release from a Core at core. The
/// release must have the minor of the Core. It must be newer than current, except at an
/// older minor over a secure connection.
fn takes(current: &Version, release: &Version, core: &Version, secure: bool) -> bool {
    minor(release) == minor(core)
        && (release > current || (secure && minor(release) != minor(current)))
}

/// Checks the manifest at endpoint for a release at the minor of core_version. Only a
/// newer release qualifies, except that a secure connection also takes a release at an
/// older minor and sets downgrade. Returns None when the endpoint serves no manifest, or
/// none for this platform. The plugin's download and install commands take the update by
/// its rid.
#[tauri::command]
pub async fn updater_check<R: Runtime>(
    webview: Webview<R>,
    endpoint: Url,
    core_version: Version,
    secure: bool,
) -> Result<Option<Metadata>, Error> {
    let updater = webview
        .updater_builder()
        .endpoints(vec![endpoint])?
        .version_comparator(move |current, release| {
            takes(&current, &release.version, &core_version, secure)
        })
        .build()?;
    let update = match updater.check().await {
        Ok(update) => update,
        // A Core answers 404 when it embeds no Console update. The plugin reports every
        // non-2xx answer as ReleaseNotFound, and a manifest with no build for this
        // platform as TargetNotFound.
        Err(Error::ReleaseNotFound | Error::TargetNotFound(_) | Error::TargetsNotFound(_)) => None,
        Err(err) => return Err(err),
    };
    let Some(update) = update else {
        return Ok(None);
    };
    let downgrade = Version::parse(&update.version)? < webview.package_info().version;
    Ok(Some(Metadata {
        current_version: update.current_version.clone(),
        version: update.version.clone(),
        downgrade,
        raw_json: update.raw_json.clone(),
        rid: webview.resources_table().add(update),
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn v(version: &str) -> Version {
        Version::parse(version).unwrap()
    }

    #[test]
    fn takes_a_newer_release_at_the_minor_of_the_core() {
        for secure in [true, false] {
            assert!(takes(&v("0.60.2"), &v("0.61.0"), &v("0.61.1"), secure));
            assert!(takes(&v("0.61.2"), &v("0.61.3"), &v("0.61.0"), secure));
        }
    }

    #[test]
    fn takes_an_older_minor_of_the_core_only_over_a_secure_connection() {
        assert!(takes(&v("0.62.0"), &v("0.61.3"), &v("0.61.3"), true));
        assert!(!takes(&v("0.62.0"), &v("0.61.3"), &v("0.61.3"), false));
    }

    #[test]
    fn rejects_an_older_patch_at_the_minor_of_the_console() {
        for secure in [true, false] {
            assert!(!takes(&v("0.61.3"), &v("0.61.2"), &v("0.61.3"), secure));
            assert!(!takes(
                &v("0.61.0"),
                &v("0.61.0-rc.1"),
                &v("0.61.0"),
                secure
            ));
        }
    }

    #[test]
    fn rejects_a_release_at_another_minor_than_the_core() {
        for secure in [true, false] {
            assert!(!takes(&v("0.61.0"), &v("0.60.0"), &v("0.61.0"), secure));
            assert!(!takes(&v("0.61.0"), &v("0.62.0"), &v("0.61.0"), secure));
            assert!(!takes(&v("0.61.0"), &v("1.61.0"), &v("0.61.0"), secure));
        }
    }

    #[test]
    fn rejects_the_running_release() {
        for secure in [true, false] {
            assert!(!takes(&v("0.61.0"), &v("0.61.0"), &v("0.61.0"), secure));
        }
    }
}
