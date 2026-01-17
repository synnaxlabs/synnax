// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useLayoutEffect } from "react";

import {
  getFromURL,
  type Platform,
  PLATFORMS,
  setInURL,
} from "@/components/platform/platform";
import { Tabs as Base, type TabsProps as BaseProps } from "@/components/Tabs";

const TABS = PLATFORMS.map(({ key, ...p }) => ({ ...p, tabKey: key }));

export interface TabsProps extends Omit<BaseProps, "tabs" | "queryParamKey"> {
  exclude?: Platform[];
  priority?: Platform[];
}

export const Tabs = ({ exclude = [], priority = [], ...rest }: TabsProps) => {
  useLayoutEffect(() => {
    const platform = getFromURL(true);
    if (platform) setInURL(platform);
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

  return <Base queryParamKey="platform" tabs={tabs} {...rest} />;
};
