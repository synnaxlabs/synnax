// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api/core";
import { check, Update } from "@tauri-apps/plugin-updater";

import { isDevBuild } from "@/platform/version/build";
import { Session } from "@/session";

// A user with update on this object may take the Console past the minor of its Core.
const RESOURCE_ID: resource.ID = { type: "builtin", key: "version" };

const MANIFEST_PATH = "/console/update/latest.json";

/**
 * An update that the Console can install. A breaking update takes the Console past the
 * minor version of the Core at coreVersion.
 */
export type Available =
  | { update: Update; breaking: false }
  | { update: Update; breaking: true; coreVersion: string };

/** The Core that the Console checks for updates. */
export interface Target {
  /** The URL of the Console update manifest that the Core serves. */
  endpoint: string;
  /** The version that the Core reports. */
  version: string;
  /** True when the user may take the Console past the minor version of the Core. */
  latestGranted: boolean;
}

/** Returns the connected Core as an update target, or null before a check reports. */
export const useTarget = (): Target | null => {
  const core = Session.Core.useSelectSelected();
  const { nodeVersion } = Synnax.useConnectionStatus().details;
  const latestGranted = Access.useUpdateGranted(RESOURCE_ID);
  if (core == null || nodeVersion == null) return null;
  const scheme = core.secure ? "https" : "http";
  return {
    endpoint: `${scheme}://${core.host}:${core.port}${MANIFEST_PATH}`,
    version: nodeVersion,
    latestGranted,
  };
};

const takesServed = (current: string, served: string): boolean =>
  !migrate.versionsEqual(current, served, { checkPatch: false }) ||
  migrate.semVerNewer(served, current);

/**
 * Finds the update that the Console offers. Synnax Desktop takes the newest release of
 * its own channel. Any other Console takes the release that its Core serves when the
 * minor versions differ or the served release is a newer patch. Next, a granted user
 * gets the newest stable release when it is past the minor version of the Core. Returns
 * null in a browser, in a dev build, and when there is no target.
 */
export const find = async (target: Target | null): Promise<Available | null> => {
  if (Session.Runtime.ENGINE !== "tauri" || (await isDevBuild())) return null;
  if (Session.Runtime.CORE_EMBEDDED) {
    const update = await check();
    return update == null ? null : { update, breaking: false };
  }
  if (target == null) return null;
  const served = await invoke<ConstructorParameters<typeof Update>[0] | null>(
    "updater_check",
    { endpoint: target.endpoint },
  );
  if (served != null) {
    const update = new Update(served);
    if (takesServed(update.currentVersion, update.version))
      return { update, breaking: false };
    await update.close();
  }
  if (!target.latestGranted) return null;
  const latest = await check();
  if (latest == null) return null;
  if (migrate.semVerNewer(latest.version, target.version, { checkPatch: false }))
    return { update: latest, breaking: true, coreVersion: target.version };
  await latest.close();
  return null;
};
