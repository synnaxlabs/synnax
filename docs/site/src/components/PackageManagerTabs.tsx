// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Tabs } from "@synnaxlabs/pluto/tabs";

const TABS = [
  {
    tabKey: "npm",
    name: "npm",
    icon: <Icon.NPM />,
  },
  {
    tabKey: "yarn",
    name: "yarn",
    icon: <Icon.Yarn />,
  },
  {
    tabKey: "pnpm",
    name: "pnpm",
    icon: <Icon.PNPM />,
  },
];

export interface PackageManagerTabsProps {
  yarn?: string;
  npm?: string;
  pnpm?: string;
}

export const PackageManagerTabs = (props: PackageManagerTabsProps): ReactElement => {
  const tabs = TABS.filter(({ tabKey }) => tabKey in props).map(
    ({ tabKey, name, icon }) => ({
      tabKey,
      name,
      icon,
    }),
  );
  const tabsProps = Tabs.useStatic({ tabs });
  return (
    <Tabs.Tabs {...tabsProps} size="large">
      {(tab) => props[tab.tabKey]}
    </Tabs.Tabs>
  );
};
