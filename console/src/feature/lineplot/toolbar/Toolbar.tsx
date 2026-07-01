// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/lineplot/toolbar/Toolbar.css";

import { lineplot } from "@synnaxlabs/client";
import { Access, Button, Flex, Icon, LinePlot, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { useExport } from "@/feature/lineplot/export";
import { Cluster } from "@/primitive/cluster";
import { CSS } from "@/primitive/css";
import { Export } from "@/primitive/export";
import { Annotations } from "@/primitive/lineplot/toolbar/Annotations";
import { Axes } from "@/primitive/lineplot/toolbar/Axes";
import { Data } from "@/primitive/lineplot/toolbar/Data";
import { Lines } from "@/primitive/lineplot/toolbar/Lines";
import { Properties } from "@/primitive/lineplot/toolbar/Properties";
import { useDownloadPlotAsCSV } from "@/primitive/lineplot/useDownloadAsCSV";
import { Toolbar as Base } from "@/primitive/toolbar";
import { Session } from "@/session";

interface Tab {
  tabKey: Session.LinePlot.ToolbarTab;
  name: string;
}

const TABS: Tab[] = [
  { tabKey: "data", name: "Data" },
  { tabKey: "lines", name: "Lines" },
  { tabKey: "axes", name: "Axes" },
  { tabKey: "properties", name: "Properties" },
  { tabKey: "annotations", name: "Rules" },
];

const Internal = (): ReactElement => {
  const key = LinePlot.useKey();
  const name = LinePlot.useSelectName();
  const dispatch = Session.useDispatch();
  const handleExport = useExport();
  const activeTab = Session.LinePlot.useSelectActiveToolbarTab();
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(key));
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      switch (tabKey) {
        case "lines":
          return <Lines />;
        case "axes":
          return <Axes />;
        case "properties":
          return <Properties />;
        case "annotations":
          return <Annotations />;
        default:
          return <Data />;
      }
    },
    [key],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(
        Session.LinePlot.setActiveToolbarTab({
          key,
          tab: tabKey as Session.LinePlot.ToolbarTab,
        }),
      );
    },
    [dispatch, key],
  );
  const downloadAsCSV = useDownloadPlotAsCSV(key);
  const value = useMemo(
    () => ({
      tabs: TABS,
      selected: activeTab,
      content,
      onSelect: handleTabSelect,
    }),
    [activeTab, content, handleTabSelect],
  );
  return (
    <Base.Content className={CSS.B("line-plot-toolbar")}>
      <Tabs.Provider value={value}>
        <Base.Header>
          <Base.Title icon={<Icon.LinePlot />}>{name}</Base.Title>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty className={CSS.BE("line-plot", "toolbar", "actions")}>
              <Button.Button
                tooltip="Download as CSV"
                sharp
                size="medium"
                variant="text"
                onClick={downloadAsCSV}
              >
                <Icon.CSV />
              </Button.Button>
              <Export.ToolbarButton onExport={() => handleExport(key)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={lineplot.ontologyID(key)}
              />
            </Flex.Box>
            {hasUpdatePermission && (
              <Tabs.Selector className={CSS.BE("line-plot", "toolbar", "selector")} />
            )}
          </Flex.Box>
        </Base.Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Base.Content>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => (
  <LinePlot.Suspended linePlotKey={layoutKey}>
    <Internal />
  </LinePlot.Suspended>
);
