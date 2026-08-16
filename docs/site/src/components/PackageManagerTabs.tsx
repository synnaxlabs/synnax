// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { type ReactElement } from "react";

const TABS = [
  { tabKey: "npm", name: "npm", icon: <Icon.NPM /> },
  { tabKey: "yarn", name: "yarn", icon: <Icon.Yarn /> },
  { tabKey: "pnpm", name: "pnpm", icon: <Icon.PNPM /> },
];

export interface PackageManagerTabsProps {
  yarn?: string;
  npm?: string;
  pnpm?: string;
}

export const PackageManagerTabs = (props: PackageManagerTabsProps): ReactElement => {
  const tabs = TABS.filter(({ tabKey }) => tabKey in props);
  return (
    <Tabs.Frame initialValue={tabs[0]?.tabKey}>
      <Tabs.Selector size="large">
        {tabs.map(({ tabKey, name, icon }) => (
          <Tabs.Tab key={tabKey} itemKey={tabKey}>
            {icon}
            {name}
          </Tabs.Tab>
        ))}
      </Tabs.Selector>
      {tabs.map(({ tabKey }) => (
        <Tabs.Content key={tabKey} itemKey={tabKey}>
          {props[tabKey as keyof PackageManagerTabsProps] as unknown as ReactElement}
        </Tabs.Content>
      ))}
    </Tabs.Frame>
  );
};
