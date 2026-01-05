// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";

export type Platform = runtime.OS | "Docker";

export interface Info {
  key: Platform;
  name: string;
  icon: Icon.ReactElement;
}

export const PLATFORMS: Info[] = [
  { key: "Linux", name: "Linux", icon: <Icon.Logo.Linux /> },
  { key: "Windows", name: "Windows", icon: <Icon.Logo.Windows /> },
  { key: "macOS", name: "macOS", icon: <Icon.Logo.Apple /> },
  { key: "Docker", name: "Docker", icon: <Icon.Logo.Docker /> },
];

export const getFromURL = (detect: boolean): Platform | null => {
  const url = new URL(window.location.href);
  const platform = url.searchParams.get("platform");
  return (
    PLATFORMS.find((p) => p.key === platform)?.key ?? (detect ? runtime.getOS() : null)
  );
};

export const setInURL = (platform: Platform) => {
  const url = new URL(window.location.href);
  url.searchParams.set("platform", platform);
  window.history.pushState({}, "", url.toString());
};
