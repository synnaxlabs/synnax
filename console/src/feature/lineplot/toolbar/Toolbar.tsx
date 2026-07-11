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
import {
  Access,
  Button,
  Flex,
  Icon,
  LinePlot,
  Panel as PlutoPanel,
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useExport } from "@/feature/lineplot/export";
import { Annotations } from "@/feature/lineplot/toolbar/Annotations";
import { Axes } from "@/feature/lineplot/toolbar/Axes";
import { Data } from "@/feature/lineplot/toolbar/Data";
import { Lines } from "@/feature/lineplot/toolbar/Lines";
import { Properties } from "@/feature/lineplot/toolbar/Properties";
import { useDownloadPlotAsCSV } from "@/feature/lineplot/useDownloadAsCSV";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

const Internal = (): ReactElement => {
  const key = LinePlot.useKey();
  const name = LinePlot.useSelectName();
  const dispatch = Session.useDispatch();
  const handleExport = useExport();
  const activeTab = Session.LinePlot.useSelectActiveToolbarTab();
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(key));
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
  return (
    <Base.Content className={CSS.B("line-plot-toolbar")}>
      <Tabs.Frame value={activeTab} onChange={handleTabSelect} grow>
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
              <Tabs.Selector className={CSS.BE("line-plot", "toolbar", "selector")}>
                <Tabs.Tab itemKey="data">Data</Tabs.Tab>
                <Tabs.Tab itemKey="lines">Lines</Tabs.Tab>
                <Tabs.Tab itemKey="axes">Axes</Tabs.Tab>
                <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
                <Tabs.Tab itemKey="annotations">Rules</Tabs.Tab>
              </Tabs.Selector>
            )}
          </Flex.Box>
        </Base.Header>
        <Tabs.Content itemKey="data">
          <Data />
        </Tabs.Content>
        <Tabs.Content itemKey="lines">
          <Lines />
        </Tabs.Content>
        <Tabs.Content itemKey="axes">
          <Axes />
        </Tabs.Content>
        <Tabs.Content itemKey="properties">
          <Properties />
        </Tabs.Content>
        <Tabs.Content itemKey="annotations">
          <Annotations />
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Content>
  );
};

export const Toolbar = (): ReactElement => {
  const { key } = PlutoPanel.useSelectTabResource();
  return (
    <LinePlot.Suspended linePlotKey={key}>
      <Internal />
    </LinePlot.Suspended>
  );
};
