// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Platform } from "@/components/platform/choice";
import { INFO } from "@/components/platform/Platform";
import { Tabs as Base, type TabsProps as BaseProps } from "@/components/tabs/Tabs";

const TABS = INFO.map(({ key, ...p }) => ({ ...p, tabKey: key }));

export interface TabsProps extends Omit<BaseProps, "tabs" | "queryParamKey"> {
  exclude?: Platform[];
  priority?: Platform[];
}

export const Tabs = ({ exclude = [], priority = [], ...rest }: TabsProps) => {
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
