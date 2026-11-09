// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import { isDev } from "@/platform/version/build";
import { type Target } from "@/platform/version/check";
import { Session } from "@/session";

const RELEASE_URL = "https://docs.synnaxlabs.com/releases/core/latest.json";
const UPGRADE_URL = "https://docs.synnaxlabs.com/reference/enterprise/upgrade";

const releaseZ = z.object({ version: migrate.semVerZ });

/**
 * Adds a status when the newest stable Core release is newer than the Core at target
 * and the user may take the Console past it. It checks again when the Core, its
 * version, or the grant changes.
 */
export const useCoreNotice = (target: Target | null): void => {
  const addStatus = Status.useAdder();
  useAsyncEffect(
    async (signal) => {
      if (
        Session.Runtime.CORE_EMBEDDED ||
        target == null ||
        !target.latestGranted ||
        isDev(target.version)
      )
        return;
      let latest: string;
      try {
        const res = await fetch(RELEASE_URL, { signal });
        latest = releaseZ.parse(await res.json()).version;
      } catch (error) {
        // A network with no internet access cannot reach the docs site.
        if (!signal.aborted) console.warn(error);
        return;
      }
      if (signal.aborted || !migrate.semVerNewer(latest, target.version)) return;
      addStatus({
        key: `coreUpdate-${latest}`,
        variant: "info",
        name: "Core",
        message: `Version ${latest} available`,
        description: `Run synnax update ${latest} as an administrator on the computer that runs the Core. In Docker, or if the command fails, follow the upgrade guide at ${UPGRADE_URL}.`,
      });
    },
    [target?.version, target?.latestGranted],
  );
};
