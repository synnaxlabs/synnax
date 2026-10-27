// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Platform } from "@/components/download/Buttons.astro";
import { appVersion } from "@/util/releases";

/** Lists the installers of a Synnax Desktop release, one per platform. */
export const installers = (version: string): Platform[] => {
  const app = appVersion(version);
  return [
    { name: "macOS", asset: `Synnax-Desktop_${app}_aarch64.dmg`, agent: /Macintosh/ },
    {
      name: "Windows",
      asset: `Synnax-Desktop_${app}_x64-setup.exe`,
      agent: /Windows NT/,
    },
  ];
};

/**
 * Finds the installer of a Synnax Desktop release for the platform of a browser.
 * @param agent - The user agent of the browser.
 * @returns The installer, or null if none installs on the platform.
 */
export const installer = (version: string, agent: string): Platform | null =>
  installers(version).find((platform) => platform.agent.test(agent)) ?? null;
