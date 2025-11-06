// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useLayoutEffect } from "react";

import {
  type Client,
  CLIENTS,
  getFromURL,
  setInURL,
} from "@/components/client/client";
import { Tabs as Core, type TabsProps as CoreProps } from "@/components/Tabs";

const TABS = CLIENTS.map(({ key, ...c }) => ({ ...c, tabKey: key }));

export interface TabsProps extends Omit<CoreProps, "tabs" | "queryParamKey"> {
  exclude?: Client[];
  priority?: Client[];
}

export const Tabs = ({ exclude = [], priority = [], ...rest }: TabsProps) => {
  useLayoutEffect(() => {
    const client = getFromURL();
    if (client) setInURL(client);
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

  return <Core queryParamKey="client" tabs={tabs} {...rest} />;
};
