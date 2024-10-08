// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/toolbar/Toolbar.css";

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { useExport } from "@/lineplot/file";
import { useSelect, useSelectToolbar } from "@/lineplot/selectors";
import { setActiveToolbarTab, type ToolbarTab } from "@/lineplot/slice";
import { Annotations } from "@/lineplot/toolbar/Annotations";
import { Axes } from "@/lineplot/toolbar/Axes";
import { Data } from "@/lineplot/toolbar/Data";
import { Lines } from "@/lineplot/toolbar/Lines";
import { Properties } from "@/lineplot/toolbar/Properties";
import { Link } from "@/link";

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
    tabKey: "properties",
    name: "Properties",
  },
];

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectToolbar();
  const linePlot = useSelect(layoutKey);
  const handleExport = useExport(name);
  const handleLink = Link.useCopyToClipboard();

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
    [layoutKey],
  );

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ tab: tabKey as ToolbarTab }));
    },
    [dispatch],
  );

  if (linePlot == null) return null;

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
          <Align.Space direction="x" align="center" empty>
            <Align.Space direction="x" empty style={{ height: "100%", width: 66 }}>
              <Button.Icon
                tooltip={`Export ${name}`}
                sharp
                size="medium"
                style={{ height: "100%" }}
                onClick={() => handleExport(linePlot.key)}
              >
                <Icon.Export />
              </Button.Icon>
              <Button.Icon
                tooltip={`Copy link to ${name}`}
                sharp
                size="medium"
                style={{ height: "100%" }}
                onClick={() =>
                  handleLink({
                    name,
                    ontologyID: { key: linePlot.key, type: "lineplot" },
                  })
                }
              >
                <Icon.Link />
              </Button.Icon>
            </Align.Space>
            <Tabs.Selector style={{ borderBottom: "none" }} />
          </Align.Space>
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Align.Space>
  );
};
