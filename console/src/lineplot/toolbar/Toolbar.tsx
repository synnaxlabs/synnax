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
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/lineplot/export";
import {
  useSelectActiveToolbarTab,
  useSelectExists,
  useSelectPendingUpload,
} from "@/lineplot/selectors";
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

const Internal = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  // The toolbar reads body fields through LinePlot selectors that throw
  // NotFoundError when the plot is not in the flux cache. Suspend until the
  // canonical record is loaded so the surrounding error boundary doesn't tear
  // the toolbar down on first mount of a freshly placed plot.
  LinePlot.useEnsureRetrieved({ key: layoutKey });
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const activeTab = useSelectActiveToolbarTab(layoutKey);
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(layoutKey));
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
          return <Annotations layoutKey={layoutKey} />;
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
              <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={lineplot.ontologyID(layoutKey)}
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

export const Toolbar = (props: ToolbarProps): ReactElement | null => {
  const exists = useSelectExists(props.layoutKey);
  const pendingUpload = useSelectPendingUpload(props.layoutKey);
  if (!exists || pendingUpload != null) return null;
  return <Internal {...props} />;
};
