// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/toolbar/Toolbar.css";

import { lineplot } from "@synnaxlabs/client";
import { Access, Button, Flex, Icon, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/lineplot/export";
import { useSelectToolbar } from "@/lineplot/selectors";
import { setActiveToolbarTab, type ToolbarTab } from "@/lineplot/slice";
import { Annotations } from "@/lineplot/toolbar/Annotations";
import { Axes } from "@/lineplot/toolbar/Axes";
import { Data } from "@/lineplot/toolbar/Data";
import { Lines } from "@/lineplot/toolbar/Lines";
import { Properties } from "@/lineplot/toolbar/Properties";
import { useDownloadPlotAsCSV } from "@/lineplot/useDownloadAsCSV";

interface Tab {
  tabKey: ToolbarTab;
  name: string;
}

const TABS: Tab[] = [
  { tabKey: "data", name: "Data" },
  { tabKey: "lines", name: "Lines" },
  { tabKey: "axes", name: "Axes" },
  { tabKey: "properties", name: "Properties" },
  { tabKey: "annotations", name: "Rules" },
];

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const state = useSelectToolbar(layoutKey);
  const hasEditPermission = Access.useUpdateGranted(lineplot.ontologyID(layoutKey));
  const handleExport = useExport();
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      switch (tabKey) {
        case "lines":
          return <Lines layoutKey={layoutKey} />;
        case "axes":
          return <Axes layoutKey={layoutKey} />;
        case "properties":
          return <Properties layoutKey={layoutKey} />;
        case "annotations":
          return <Annotations linePlotKey={layoutKey} />;
        default:
          return <Data layoutKey={layoutKey} />;
      }
    },
    [layoutKey],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ key: layoutKey, tab: tabKey as ToolbarTab }));
    },
    [dispatch, layoutKey],
  );
  const downloadAsCSV = useDownloadPlotAsCSV(layoutKey);
  const value = useMemo(
    () => ({
      tabs: TABS,
      selected: state?.activeTab,
      content,
      onSelect: handleTabSelect,
    }),
    [state?.activeTab, content, handleTabSelect],
  );
  if (state == null) return null;
  return (
    <Core.Content className={CSS.B("line-plot-toolbar")}>
      <Tabs.Provider value={value}>
        <Core.Header>
          <Core.Title icon={<Icon.LinePlot />}>{name}</Core.Title>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty style={{ height: "100%", width: 86 }}>
              <Button.Button
                tooltip="Download as CSV"
                sharp
                size="medium"
                variant="text"
                onClick={downloadAsCSV}
              >
                <Icon.CSV />
              </Button.Button>
              <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={lineplot.ontologyID(layoutKey)}
              />
            </Flex.Box>
            {hasEditPermission && <Tabs.Selector style={{ borderBottom: "none" }} />}
          </Flex.Box>
        </Core.Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Core.Content>
  );
};
