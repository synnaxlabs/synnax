import { memo, useCallback, useEffect, useState } from "react";

import type { Channel, ChannelPayload } from "@synnaxlabs/client";
import { Input, Space, Header, Tabs, Select } from "@synnaxlabs/pluto";
import type { SelectMultipleProps, SelectProps } from "@synnaxlabs/pluto";
import { HiChartBar } from "react-icons/hi";
import { useDispatch } from "react-redux";

import {
  setVisualization,
  updateVisualization,
  useSelectSugaredVisualization,
  useSelectVisualization,
} from "../store";
import { LinePlotVisualization } from "../types";

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

  const vis = useSelectSugaredVisualization<LinePlotVisualization>();

  const ranges = useSelectRanges();

  const dispatch = useDispatch();

  const [channels, setChannelOpts] = useState<Array<ChannelPayload & { key: string }>>(
    []
  );

  useAsyncEffect(async () => {
    if (client == null) return;
    const channels = await client.channel.retrieveAll();
    setChannelOpts(
      channels.map((ch) => ch.payload as ChannelPayload & { key: string })
    );
  }, [client]);

  if (vis == null) return null;

  const handleChannelChange = useCallback(
    (key: string, value: any): void => {
      dispatch(updateVisualization({ key: vis.key, channels: { [key]: value } }));
    },
    [dispatch]
  );

  return (
    <Space style={{ padding: "2rem", maxWidth: "100%" }}>
      <YAxisSelect
        number={1}
        data={channels}
        onChange={handleChannelChange}
        value={vis.channels.y1}
      />
      {/* <YAxisSelect
        number={2}
        data={channels}
        onChange={handleChannelChange}
        value={vis.channels.y2}
      />
      <YAxisSelect
        number={3}
        data={channels}
        onChange={handleChannelChange}
        value={vis.channels.y3}
      />
      <YAxisSelect
        number={4}
        data={channels}
        onChange={handleChannelChange}
        value={vis.channels.y4}
      />
      <Space direction="horizontal" grow>
        <XAxisSelect
          number={1}
          data={channels}
          onChange={handleChannelChange}
          value={vis.channels.x1}
        />
        <RangeSelect
          number={1}
          data={ranges}
          onChange={handleChannelChange}
          value={vis.ranges.x1}
        />
      </Space>
      <Space direction="horizontal" grow>
        <XAxisSelect
          number={2}
          data={channels}
          onChange={handleChannelChange}
          value={vis.channels.x2}
        />
        <RangeSelect
          number={2}
          data={ranges}
          onChange={handleChannelChange}
          value={vis.ranges.x2}
        />
      </Space> */}
    </Space>
  );
};

interface YAxisSelectProps {
  number: number;
  data: ChannelPayload[];
  onChange: (key: string, v: readonly string[]) => void;
  value: readonly string[];
}

const YAxisSelect = memo(
  ({ number, data, onChange, value }: YAxisSelectProps): JSX.Element => (
    <Input.Item<readonly string[], SelectMultipleProps<ChannelPayload>>
      direction="horizontal"
      label={`Y${number}:`}
      data={data}
      tagKey="name"
      onChange={(v) => onChange(`y${number}`, v)}
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
  number: number;
  data: ChannelPayload[];
  onChange: (key: string, v: string) => void;
  value: string;
}

const XAxisSelect = memo(
  ({ number, data, onChange, value }: XAxisSelectProps): JSX.Element => (
    <Input.Item<string, SelectProps<ChannelPayload>>
      direction="horizontal"
      tagKey="name"
      label={`X${number}:`}
      data={data}
      onChange={(v) => onChange(`x${number}`, v)}
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
  number: number;
  data: Range[];
  onChange: (key: string, v: readonly string[]) => void;
  value: readonly string[];
}

const RangeSelect = memo(
  ({ number, data, onChange, value }: RangeSelectProps): JSX.Element => (
    <Input.Item<readonly string[], SelectMultipleProps<Range>>
      direction="horizontal"
      tagKey="name"
      label={`X${number} Ranges:`}
      data={data}
      onChange={(v: readonly string[]) => onChange(`range${number}`, v)}
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
