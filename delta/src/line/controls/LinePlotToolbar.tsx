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

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { LinePlotChannelControls } from "@/line/controls/LinePlotChannelControls";

export interface LinePlotToolbarProps {
  layoutKey: string;
}

const TABS = [
  {
    tabKey: "channels",
    name: "Channels",
  },
];

export const LinePlotToolBar = ({ layoutKey }: LinePlotToolbarProps): ReactElement => {
  const content = useCallback(
    ({ tabKey }: Tab): ReactElement => (
      <LinePlotChannelControls layoutKey={layoutKey} />
    ),
    [layoutKey]
  );

  const tabProps = Tabs.useStatic({ tabs: TABS, content });

  return (
    <Space empty>
      <Tabs.Provider value={tabProps}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Visualize />}>LinePlot</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};
