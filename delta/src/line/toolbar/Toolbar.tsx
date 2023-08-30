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
import { Layout } from "@/layout";
import { useSelectLineToolbar } from "@/line/selectors";
import { LineToolbarTab, setLineActiveToolbarTab } from "@/line/slice";
import { Annotations } from "@/line/toolbar/Annotations";
import { Axes } from "@/line/toolbar/Axes";
import { Data } from "@/line/toolbar/Data";
import { Lines } from "@/line/toolbar/Lines";
import { Properties } from "@/line/toolbar/Properties";

import "@/line/toolbar/Toolbar.css";

export interface ToolbarProps {
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

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const { name } = Layout.useSelectRequired(layoutKey);
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
