// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Tabs } from "@synnaxlabs/pluto/tabs";

const TABS = [
  {
    tabKey: "python",
    name: "Python",
    icon: <Icon.Python />,
  },
  {
    tabKey: "typescript",
    name: "Typescript",
    icon: <Icon.Typescript />,
  },
];

export const ClientTabs = (props: any): ReactElement => {
  const tabsProps = Tabs.useStatic({ tabs: TABS });
  return (
    <Tabs.Tabs {...tabsProps}>
      {(tab) => (
        <div>
          <h2>Using {tab.name}</h2>
          {props[`setup-${tab.tabKey}`]}
          {props[tab.tabKey]}
        </div>
      )}
    </Tabs.Tabs>
  );
};
