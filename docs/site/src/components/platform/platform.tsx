// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type Icon as PIcon } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";
import { z } from "zod";

export const platformZ = z.enum(["docker", "linux", "macos", "windows"]);
export type Platform = z.infer<typeof platformZ>;

export interface PlatformInfo {
  key: Platform;
  name: string;
  icon: PIcon.Element;
}

export const PLATFORMS: PlatformInfo[] = [
  { key: "linux", name: "Linux", icon: <Icon.Logo.Linux /> },
  { key: "windows", name: "Windows", icon: <Icon.Logo.Windows /> },
  { key: "macos", name: "macOS", icon: <Icon.Logo.Apple /> },
  { key: "docker", name: "Docker", icon: <Icon.Logo.Docker /> },
];

export const getPlatformFromURL = (detect: boolean): Platform | null => {
  const url = new URL(window.location.href);
  const platform = url.searchParams.get("platform");
  return (
    platformZ.safeParse(platform).data ??
    (detect ? (platformZ.safeParse(runtime.getOS()).data ?? null) : null)
  );
};

export const setPlatformInURL = (platform: Platform) => {
  const url = new URL(window.location.href);
  url.searchParams.set("platform", platform);
  window.history.pushState({}, "", url.toString());
};
