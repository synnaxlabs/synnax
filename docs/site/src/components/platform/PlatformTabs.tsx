// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useLayoutEffect } from "react";

import {
  getPlatformFromURL,
  type Platform,
  PLATFORMS,
  setPlatformInURL,
} from "@/components/platform/platform";
import { Tabs } from "@/components/Tabs";

const TABS = PLATFORMS.map(({ key, ...p }) => ({ ...p, tabKey: key }));

export const PlatformTabs = ({
  exclude = [] as Platform[],
  priority = [] as Platform[],
  ...props
}) => {
  useLayoutEffect(() => {
    const platform = getPlatformFromURL(true);
    if (platform) setPlatformInURL(platform);
  }, []);

  const excludeSet = new Set(exclude);
  const tabs = TABS.filter((tab) => !excludeSet.has(tab.tabKey));

  if (priority.length > 0)
    tabs.sort((a, b) => {
      let aIndex = priority.indexOf(a.tabKey);
      if (aIndex === -1) aIndex = priority.length;
      let bIndex = priority.indexOf(b.tabKey);
      if (bIndex === -1) bIndex = priority.length;
      return aIndex - bIndex;
    });

  return <Tabs queryParamKey="platform" tabs={tabs} {...props} />;
};
