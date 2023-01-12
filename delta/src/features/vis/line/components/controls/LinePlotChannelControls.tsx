import { useCallback, useEffect, useState } from "react";

import { ChannelPayload } from "@synnaxlabs/client";
import { Space } from "@synnaxlabs/pluto";

import { SelectAxisInputItem, SelectMultipleAxesInputItem } from "../../../components";
import { AxisKey } from "../../../types";

import { ControlledLineVisProps } from "./types";

import { useClusterClient } from "@/features/cluster";
import { useSelectRanges, SelectMultipleRangesInputItem } from "@/features/workspace";
import { useAsyncEffect } from "@/hooks";

export const LinePlotChannelControls = ({
  vis,
  setVis,
}: ControlledLineVisProps): JSX.Element | null => {
  const ranges = useSelectRanges();

  const client = useClusterClient();

  const [channels, setChannels] = useState<ChannelPayload[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const ch = await client.channel.retrieveAll();
    setChannels(ch.map((ch) => ch.payload));
  }, [client]);

  const handleChannelSelect = useCallback(
    (key: AxisKey, value: readonly string[] | string): void => {
      setVis({ channels: { [key]: value } });
    },
    [setVis]
  );

  const handleRangeSelect = useCallback(
    (value: readonly string[]): void => {
      setVis({ ranges: { x1: value } });
    },
    [setVis]
  );

  return (
    <Space style={{ padding: "2rem", maxWidth: "100%" }}>
      <SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleChannelSelect}
        value={vis.channels.y1}
        data={channels}
        grow
      />
      <Space direction="horizontal" grow>
        <SelectAxisInputItem
          axis={"x1"}
          onChange={handleChannelSelect}
          value={vis.channels.x1}
          data={channels}
          style={{ width: "25%" }}
        />
        <SelectMultipleRangesInputItem
          data={ranges}
          onChange={handleRangeSelect}
          value={vis.ranges.x1.map((v) => v.key)}
          style={{ width: "75%" }}
        />
      </Space>
    </Space>
  );
};
