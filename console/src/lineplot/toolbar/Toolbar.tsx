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
import { Access, Button, Flex, Icon, LinePlot, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { useExport } from "@/lineplot/export";
import { type ToolbarTab } from "@/lineplot/session/slice";
import { Tab } from "@/lineplot/tab";
import { Annotations } from "@/lineplot/toolbar/Annotations";
import { Axes } from "@/lineplot/toolbar/Axes";
import { Data } from "@/lineplot/toolbar/Data";
import { Lines } from "@/lineplot/toolbar/Lines";
import { Properties } from "@/lineplot/toolbar/Properties";
import { useDownloadPlotAsCSV } from "@/lineplot/useDownloadAsCSV";

import { Session } from "../session";

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

const Internal = (): ReactElement | null => {
  const key = LinePlot.useKey();
  const name = LinePlot.useSelectName({});
  const activeTab = Session.useSelectSelectedToolbarTab();
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(key));
  const handleExport = useExport();
  const content = useCallback(({ tabKey }: Tabs.Tab) => {
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
  }, []);
  const handleTabSelect = Session.useSetSelectedToolbarTab();
  const downloadAsCSV = useDownloadPlotAsCSV(key);
  const value = useMemo(
    () =>
      ({
        tabs: TABS,
        selected: activeTab,
        content,
        onSelect: handleTabSelect,
      }) as Tabs.ContextValue,
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

export const Toolbar = (): ReactElement | null => {
  const { key } = Tab.useArgs();
  return (
    <LinePlot.Suspended lineplotKey={key}>
      <Internal />
    </LinePlot.Suspended>
  );
};
