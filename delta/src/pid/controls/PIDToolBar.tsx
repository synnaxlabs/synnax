// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { Icon } from "@synnaxlabs/media";
import { Space, Tab, Tabs } from "@synnaxlabs/pluto";

import { PIDElementPropertiesControls } from "./PIDElementPropertiesControls";
import { PIDElements } from "./PIDElementsControls";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { useSelectRequiredLayout } from "@/layout";

export interface PIDToolbarProps {
  layoutKey: string;
}

const TABS = [
  {
    tabKey: "elements",
    name: "Elements",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },
];

export const PIDToolbar = ({ layoutKey }: PIDToolbarProps): ReactElement => {
  const { name } = useSelectRequiredLayout(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tab): ReactElement => {
      switch (tabKey) {
        case "elements":
          return <PIDElements layoutKey={layoutKey} />;
        default:
          return <PIDElementPropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey]
  );

  const tabsProps = Tabs.useStatic({
    tabs: TABS,
    content,
  });

  return (
    <Space empty>
      <Tabs.Provider value={tabsProps}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Control />}>{name}</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};
