// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, useCallback, useState } from "react";

import { Channel, KeyedChannelPayload } from "@synnaxlabs/client";
import { Input, Space, Header, Tabs, Select } from "@synnaxlabs/pluto";
import type { SelectMultipleProps, SelectProps } from "@synnaxlabs/pluto";
import { HiChartBar } from "react-icons/hi";

import { useUpdateVisualization } from "../../hooks";
import { AxisKey, LinePlotVS, XAxisKey, YAxisKey } from "../../line/types";
import { useSelectSugaredVisualization } from "../../store";

import { useClusterClient } from "@/features/cluster";
import { Range, useSelectRanges } from "@/features/workspace";
import { useAsyncEffect } from "@/hooks";

const VisualizationIcon = HiChartBar;

const Content = (): JSX.Element => {
  const props = Tabs.useStatic({
    tabs: [
      {
        tabKey: "channels",
        title: "Channels",
        content: <LinePlotChannelControls />,
      },
      {
        tabKey: "annotations",
        title: "Annotations",
        content: <h1>Annotations</h1>,
      },
      {
        tabKey: "styles",
        title: "Styles",
        content: <h1>Styles</h1>,
      },
    ],
  });
  return (
    <Space>
      <Tabs.Provider value={props}>
        <Header level="h4" divided>
          <Header.Title startIcon={<VisualizationIcon />}>Visualization</Header.Title>
          <Tabs.Selector style={{ borderBottom: "none" }} />
        </Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};

export const VisualizationToolBar = {
  key: "visualization",
  content: <Content />,
  icon: <VisualizationIcon />,
  minSize: 150,
  maxSize: 500,
};
