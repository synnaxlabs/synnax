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

import { useUpdateVisualization } from "../hooks";
import { AxisKey, LinePlotVS, XAxisKey, YAxisKey } from "../line/types";
import { useSelectSugaredVisualization } from "../store";

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

const LinePlotChannelControls = (): JSX.Element | null => {
  const client = useClusterClient();

  const vis = useSelectSugaredVisualization<LinePlotVS>();

  const ranges = useSelectRanges();

  const updateV = useUpdateVisualization(vis?.key ?? "");

  const [channels, setChannelOpts] = useState<KeyedChannelPayload[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const channels = await client.channel.retrieveAll();
    setChannelOpts(channels.map((c) => c.payload as KeyedChannelPayload));
  }, [client]);

  const handleYSelect = useCallback(
    (key: YAxisKey, value: readonly string[]): void => {
      updateV({ channels: { [key]: value } });
    },
    [updateV]
  );

  const handleXSelect = useCallback(
    (key: XAxisKey, value: string): void => {
      updateV({ channels: { [key]: value } });
    },
    [updateV]
  );

  const handleRangeSelect = useCallback(
    (key: AxisKey, value: readonly string[]): void => {
      updateV({ ranges: { [key]: value } });
    },
    [updateV]
  );

  if (vis == null) return null;

  return (
    <Space style={{ padding: "2rem", maxWidth: "100%" }}>
      <YAxisSelect
        axis={"y1"}
        data={channels}
        onChange={handleYSelect}
        value={vis.channels.y1}
      />
      <Space direction="horizontal" grow>
        <XAxisSelect
          axis={"x1"}
          data={channels}
          onChange={handleXSelect}
          value={vis.channels.x1}
        />
        <RangeSelect
          axis={"x1"}
          data={ranges}
          onChange={handleRangeSelect}
          value={vis.ranges.x1.map((v) => v.key)}
        />
      </Space>
    </Space>
  );
};

interface YAxisSelectProps {
  axis: YAxisKey;
  data: Channel[];
  onChange: (key: YAxisKey, v: readonly string[]) => void;
  value: readonly string[];
}

const YAxisSelect = memo(
  ({ axis, data, onChange, value }: YAxisSelectProps): JSX.Element => (
    <Input.Item<readonly string[], SelectMultipleProps<Channel>>
      direction="horizontal"
      label={`${axis}:`}
      data={data}
      tagKey="name"
      onChange={(v) => onChange(axis, v)}
      value={value}
      style={{ width: "100%" }}
      columns={[
        {
          key: "name",
          label: "Name",
        },
      ]}
    >
      {Select.Multiple}
    </Input.Item>
  )
);
YAxisSelect.displayName = "YAxisSelect";

interface XAxisSelectProps {
  axis: XAxisKey;
  data: Channel[];
  onChange: (key: XAxisKey, v: string) => void;
  value: string;
}

const XAxisSelect = memo(
  ({ axis, data, onChange, value }: XAxisSelectProps): JSX.Element => (
    <Input.Item<string, SelectProps<Channel>>
      direction="horizontal"
      tagKey="name"
      label={`${axis}:`}
      data={data}
      onChange={(v) => onChange(axis, v)}
      grow
      value={value}
      columns={[
        {
          key: "name",
          label: "Name",
        },
      ]}
    >
      {Select}
    </Input.Item>
  )
);
XAxisSelect.displayName = "XAxisSelect";

interface RangeSelectProps {
  axis: XAxisKey;
  data: Range[];
  onChange: (key: XAxisKey, v: readonly string[]) => void;
  value: readonly string[];
}

const RangeSelect = memo(
  ({ axis, data, onChange, value }: RangeSelectProps): JSX.Element => (
    <Input.Item<readonly string[], SelectMultipleProps<Range>>
      direction="horizontal"
      tagKey="name"
      label={`${axis} Ranges:`}
      data={data}
      onChange={(v: readonly string[]) => onChange(axis, v)}
      grow
      value={value}
      columns={[
        {
          key: "name",
          label: "Name",
        },
      ]}
    >
      {Select.Multiple}
    </Input.Item>
  )
);
RangeSelect.displayName = "RangeSelect";
