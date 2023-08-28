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
import { Align, Tabs } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { useSelectRequiredLayout } from "@/layout";
import { Data } from "@/line/controls/Data";

import { useSelectLineToolbar } from "../store/selectors";
import { LineToolbarTab, setLineActiveToolbarTab } from "../store/slice";

import { Annotations } from "./Annotations";
import { Axes } from "./Axes";
import { Lines } from "./Lines";
import { Properties } from "./Properties";

import "@/line/controls/LinePlotToolbar.css";

export interface LinePlotToolbarProps {
  layoutKey: string;
}

const TABS = [
  {
    tabKey: "data",
    name: "Data",
  },
  {
    tabKey: "lines",
    name: "Lines",
  },
  {
    tabKey: "axes",
    name: "Axes",
  },
  {
    tabKey: "annotations",
    name: "Annotations",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },
];

export const LinePlotToolBar = ({ layoutKey }: LinePlotToolbarProps): ReactElement => {
  const { name } = useSelectRequiredLayout(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectLineToolbar();
  const content = useCallback(
    ({ tabKey }: Tabs.Tab): ReactElement => {
      switch (tabKey) {
        case "lines":
          return <Lines layoutKey={layoutKey} />;
        case "axes":
          return <Axes layoutKey={layoutKey} />;
        case "properties":
          return <Properties layoutKey={layoutKey} />;
        case "annotations":
          return <Annotations layoutKey={layoutKey} />;
        default:
          return <Data layoutKey={layoutKey} />;
      }
    },
    [layoutKey]
  );

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setLineActiveToolbarTab({ tab: tabKey as LineToolbarTab }));
    },
    [dispatch]
  );

  return (
    <Align.Space empty className={CSS.B("line-plot-toolbar")}>
      <Tabs.Provider
        value={{
          tabs: TABS,
          selected: toolbar.activeTab,
          content,
          onSelect: handleTabSelect,
        }}
      >
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Visualize />}>{name}</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Align.Space>
  );
};
