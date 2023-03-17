// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { ChannelPayload } from "@synnaxlabs/client";
import { Space } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useClusterClient } from "@/cluster";
import { useAsyncEffect } from "@/hooks";
import { AxisKey } from "@/vis/axis";
import { SelectAxisInputItem, SelectMultipleAxesInputItem } from "@/vis/components";
import { Channels } from "@/vis/line/channels";
import { Ranges } from "@/vis/line/ranges";
import { updateVis } from "@/vis/store";
import { SelectMultipleRangesInputItem, useSelectRanges } from "@/workspace";

export interface LinePlotChannelControlsProps {
  layoutKey: string;
}

export const LinePlotChannelControls = ({
  layoutKey,
}: LinePlotChannelControlsProps): JSX.Element | null => {
  const ranges = Ranges.useSelectCore(layoutKey);
  const channels = Channels.useSelectCore(layoutKey);
  const dispatch = useDispatch();
  const allRanges = useSelectRanges();

  const client = useClusterClient();

  const [allChannels, setAllChannels] = useState<ChannelPayload[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const ch = await client.channels.retrieveAll();
    setAllChannels(ch.map((ch) => ch.payload));
  }, [client]);

  const handleChannelSelect = (
    key: AxisKey,
    value: readonly string[] | string
  ): void => {
    dispatch(updateVis({ key: layoutKey, channels: { [key]: value } }));
  };

  const handleRangeSelect = (value: readonly string[]): void => {
    dispatch(updateVis({ key: layoutKey, ranges: { x1: value } }));
  };

  return (
    <Space style={{ padding: "2rem", width: "100%" }}>
      <SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleChannelSelect}
        value={channels.y1}
        data={allChannels}
        location="top"
        grow
      />
      <SelectMultipleAxesInputItem
        axis={"y2"}
        onChange={handleChannelSelect}
        value={channels.y2}
        data={allChannels}
        location="top"
        grow
      />
      <Space direction="x" grow wrap>
        <SelectMultipleRangesInputItem
          data={allRanges}
          onChange={handleRangeSelect}
          value={ranges.x1}
          location="top"
          grow
        />

        <SelectAxisInputItem
          axis={"x1"}
          onChange={handleChannelSelect}
          value={channels.x1}
          location="top"
          data={allChannels}
          grow
        />
      </Space>
    </Space>
  );
};
