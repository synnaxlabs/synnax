// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Tabs } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { useSelectToolbar } from "@/table/selectors";
import { type ToolbarTab, setActiveToolbarTab } from "@/table/slice";
import { Properties } from "@/table/toolbar/Properties";
import { Shape } from "@/table/toolbar/Shape";

const TABS = [
  {
    tabKey: "shape",
    name: "Shape",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },
];

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const d = useDispatch();
  const { name } = Layout.useSelectRequired(layoutKey);
  const toolbar = useSelectToolbar();
  const content = useCallback(
    ({ tabKey }: Tabs.Tab): ReactElement => {
      switch (tabKey) {
        case "properties":
          return <Properties layoutKey={layoutKey} />;
        default:
          return <Shape layoutKey={layoutKey} />;
      }
    },
    [layoutKey],
  );

  const handleTabSelect = useCallback(
    (selected: string) => {
      d(setActiveToolbarTab({ tab: selected as ToolbarTab }));
    },
    [d],
  );

  return (
    <Align.Space empty style={{ height: "100%" }}>
      <Tabs.Provider
        value={{
          tabs: TABS,
          selected: toolbar.activeTab,
          onSelect: handleTabSelect,
          content,
        }}
      >
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.PID />}>{name}</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
      </Tabs.Provider>
    </Align.Space>
  );
};
