// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, status } from "@synnaxlabs/client";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";
import { invoke } from "@tauri-apps/api/core";
import { check, Update } from "@tauri-apps/plugin-updater";

import { isDev, isDevBuild } from "@/platform/version/build";
import { Session } from "@/session";

// A user with update on this object may take the Console past the minor of its Core.
const RESOURCE_ID: resource.ID = { type: "builtin", key: "version" };

const MANIFEST_PATH = "/console/update/latest.json";

export const STATUS_KEY_PREFIX = "versionUpdate";

/**
 * An update that the Console can install. A downgrade installs an older Console. A
 * breaking update takes the Console past the minor version of the Core at coreVersion.
 */
export type Available =
  | { update: Update; variant: "upgrade" | "downgrade" }
  | { update: Update; variant: "breaking"; coreVersion: string };

/** A downgrade that the Console refuses over an insecure connection to the Core. */
export interface Blocked {
  variant: "blocked";
  status: status.Status;
}

type Served = ConstructorParameters<typeof Update>[0] & { downgrade: boolean };

/** The Core that the Console checks for updates. */
export interface Target {
  /** The URL of the Console update manifest that the Core serves. */
  endpoint: string;
  /** The version that the Core reports. */
  version: string;
  /** True when the Console connects to the Core over TLS. */
  secure: boolean;
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
    secure: core.secure,
    latestGranted,
  };
};

/**
 * Finds the update that the Console offers, or null. The update is breaking when it
 * takes a granted user past the minor version of the Core. Returns a Blocked when the
 * Core is at an older minor version and the connection to it is not secure.
 */
export const find = async (
  target: Target | null,
): Promise<Available | Blocked | null> => {
  if (Session.Runtime.ENGINE !== "tauri" || (await isDevBuild())) return null;
  if (Session.Runtime.CORE_EMBEDDED) {
    const update = await check();
    return update == null ? null : { update, variant: "upgrade" };
  }
  if (target == null) return null;
  const served = await invoke<Served | null>("updater_check", {
    endpoint: target.endpoint,
    coreVersion: target.version,
    secure: target.secure,
  });
  if (served != null)
    return {
      update: new Update(served),
      variant: served.downgrade ? "downgrade" : "upgrade",
    };
  if (isDev(target.version)) return null;
  if (!target.secure) {
    const current = await getVersion();
    if (migrate.semVerOlder(target.version, current, { checkPatch: false }))
      return {
        variant: "blocked",
        status: status.create({
          variant: "warning",
          name: "Console",
          message: "Cannot downgrade over an insecure connection",
          description: `Core ${target.version} is older than Console ${current}. Install Console ${target.version} by hand, or connect to the Core over TLS to downgrade.`,
        }),
      };
  }
  if (!target.latestGranted) return null;
  let latest: Update | null;
  try {
    latest = await check();
  } catch (error) {
    // A network with no internet access cannot reach the docs site.
    console.warn(error);
    return null;
  }
  if (latest == null) return null;
  if (migrate.semVerNewer(latest.version, target.version, { checkPatch: false }))
    return { update: latest, variant: "breaking", coreVersion: target.version };
  await latest.close();
  return null;
};
