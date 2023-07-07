// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { ChannelKey, ChannelKeys } from "@synnaxlabs/client";
import { Space } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelectLinePlot } from "@/line/store/selectors";
import {
  setLinePlotRanges,
  setLinePlotXChannel,
  setLinePlotYChannels,
} from "@/line/store/slice";
import { AxisKey, XAxisKey, YAxisKey } from "@/vis/axis";
import { SelectAxisInputItem, SelectMultipleAxesInputItem } from "@/vis/components";
import { SelectMultipleRangesInputItem, useSelectRanges } from "@/workspace";

export interface LinePlotDataControlsProps {
  layoutKey: string;
}

export const LinePlotDataControls = ({
  layoutKey,
}: LinePlotDataControlsProps): ReactElement | null => {
  const vis = useSelectLinePlot(layoutKey);
  const dispatch = useDispatch();
  const allRanges = useSelectRanges();

  const handleYChannelSelect = useCallback(
    (key: AxisKey, value: readonly ChannelKey[]): void => {
      dispatch(
        setLinePlotYChannels({
          key: layoutKey,
          axisKey: key as YAxisKey,
          channels: value as ChannelKeys,
        })
      );
    },
    [dispatch, layoutKey]
  );

  const handleXChannelSelect = useCallback(
    (key: AxisKey, value: ChannelKey): void => {
      dispatch(
        setLinePlotXChannel({ key: layoutKey, axisKey: key as XAxisKey, channel: value })
      );
    },
    [dispatch, layoutKey]
  );

  const handleRangeSelect = (key: XAxisKey, value: readonly string[]): void => {
    dispatch(setLinePlotRanges({ key: layoutKey, axisKey: key, ranges: value as string[] }));
  };

  return (
    <Space style={{ padding: "2rem", width: "100%" }}>
      <SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleYChannelSelect}
        value={vis.channels.y1}
        location="top"
        grow
      />
      <SelectMultipleAxesInputItem
        axis={"y2"}
        onChange={handleYChannelSelect}
        value={vis.channels.y2}
        location="top"
        grow
      />
      <Space direction="x" grow wrap>
        <SelectMultipleRangesInputItem
          data={allRanges}
          onChange={(v) => handleRangeSelect("x1", v)}
          value={vis.ranges.x1}
          location="top"
          grow
        />

        <SelectAxisInputItem
          axis={"x1"}
          onChange={handleXChannelSelect}
          value={vis.channels.x1}
          location="top"
          grow
        />
      </Space>
    </Space>
  );
};
