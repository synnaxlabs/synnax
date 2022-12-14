// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Synnax } from "@synnaxlabs/client";
import type { ChannelPayload } from "@synnaxlabs/client";
import { Select, Space } from "@synnaxlabs/pluto";

import { LinePlotVisualization, SugaredLinePlotVisualization } from "../../types";

import { useSelectRanges } from "@/features/workspace";
import { useAsyncEffect } from "@/hooks";

export interface LinePlotControlsProps {
  visualization: SugaredLinePlotVisualization;
  onChange: (vis: LinePlotVisualization) => void;
  client: Synnax;
}

export const LinePlotControls = ({
  visualization,
  onChange,
  client,
}: LinePlotControlsProps): JSX.Element => {
  const ranges = useSelectRanges();
  const { channels } = visualization;
  const [channelOpts, setChannelOpts] = useState<
    Array<ChannelPayload & { key: string }>
  >([]);

  useAsyncEffect(async () => {
    const channels = await client.channel.retrieveAll();
    setChannelOpts(
      channels.map((ch) => ch.payload as ChannelPayload & { key: string })
    );
  }, [client]);

  const handleChannelSelect = (selected: string[]): void => {
    onChange({
      ...visualization,
      ranges: visualization.ranges.map((range) => range.key),
      channels: selected,
    });
  };

  const handleRangeSelect = (selected: string[]): void => {
    onChange({
      ...visualization,
      ranges: selected,
      channels: visualization.channels,
    });
  };

  return (
    <Space direction="vertical">
      <Select.Multiple
        selected={channels}
        onSelect={handleChannelSelect}
        options={
          channelOpts as unknown as Array<Record<string, string> & { key: string }>
        }
        tagKey="name"
        listPosition="top"
        columns={[
          {
            key: "name",
            label: "Name",
          },
        ]}
      />
      <Select.Multiple
        selected={visualization.ranges.map((range) => range.key)}
        listPosition="top"
        onSelect={handleRangeSelect}
        options={ranges}
        columns={[
          {
            key: "name",
            label: "Name",
          },
        ]}
      />
    </Space>
  );
};
