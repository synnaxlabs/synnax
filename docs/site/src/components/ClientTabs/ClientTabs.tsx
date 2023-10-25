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
  const handleSelect = (tab: string) => {
    // set the tab key as a query param on url
    // this will allow us to persist the tab selection
    // when navigating between pages
    console.log(window.location)
    const currHref = window.location.href;
    const paramIdx = currHref.indexOf("tab=");
    let paramEndIdx = currHref.indexOf("&", paramIdx);
    const newHref = paramIdx === -1
      ? `${currHref}&tab=${tab}`
      : `${currHref.substring(0, paramIdx)}tab=${tab}${paramEndIdx === -1 ? "" : currHref.substring(paramEndIdx)}`;
    window.history.replaceState(
      {},
      "",
      newHref,
    );
  }


  const tabsProps = Tabs.useStatic({ tabs: TABS, onSelect: handleSelect });
  return (
    <Tabs.Tabs {...tabsProps} size="large">
      {(tab) => (
        <div>
          {props[`setup-${tab.tabKey}`]}
          {props[tab.tabKey]}
        </div>
      )}
    </Tabs.Tabs>
  );
};
