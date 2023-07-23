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
import { useDispatch } from "react-redux";

import { useSelectPIDToolbar } from "../store/selectors";
import { PIDToolbarTab, setPIDActiveToolbarTab } from "../store/slice";

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
  const dispatch = useDispatch();
  const toolbar = useSelectPIDToolbar();
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

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setPIDActiveToolbarTab({ tab: tabKey as PIDToolbarTab }));
    },
    [dispatch]
  );

  return (
    <Space empty>
      <Tabs.Provider
        value={{
          tabs: TABS,
          selected: toolbar.activeTab,
          onSelect: handleTabSelect,
          content,
        }}
      >
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Control />}>{name}</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};
