// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { type channel } from "@synnaxlabs/client";
import { Align } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelect } from "@/lineplot/selectors";
import { setRanges, setXChannel, setYChannels } from "@/lineplot/slice";
import { Range } from "@/range";
import { useSelectMultiple } from "@/range/selectors";
import { Vis } from "@/vis";
import { type AxisKey, type XAxisKey, type YAxisKey } from "@/vis/axis";

export interface DataProps {
  layoutKey: string;
}

export const Data = ({ layoutKey }: DataProps): ReactElement | null => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();
  const allRanges = useSelectMultiple();

  const handleYChannelSelect = useCallback(
    (key: AxisKey, value: readonly channel.Key[]): void => {
      dispatch(
        setYChannels({
          key: layoutKey,
          axisKey: key as YAxisKey,
          channels: value as channel.Keys,
        }),
      );
    },
    [dispatch, layoutKey],
  );

  const handleXChannelSelect = useCallback(
    (key: AxisKey, value: channel.Key): void => {
      dispatch(
        setXChannel({
          key: layoutKey,
          axisKey: key as XAxisKey,
          channel: value,
        }),
      );
    },
    [dispatch, layoutKey],
  );

  const handleRangeSelect = (key: XAxisKey, value: string[]): void => {
    dispatch(setRanges({ key: layoutKey, axisKey: key, ranges: value }));
  };

  return (
    <Align.Space style={{ padding: "2rem", width: "100%" }}>
      <Vis.SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleYChannelSelect}
        value={vis.channels.y1}
        align="center"
        grow
      />
      <Vis.SelectMultipleAxesInputItem
        axis={"y2"}
        onChange={handleYChannelSelect}
        value={vis.channels.y2}
        grow
      />
      <Align.Space direction="x" grow wrap>
        <Range.SelectMultipleInputItem
          data={allRanges}
          onChange={(v) => handleRangeSelect("x1", v)}
          value={vis.ranges.x1}
          grow
        />
        <Vis.SelectAxisInputItem
          axis={"x1"}
          onChange={handleXChannelSelect}
          value={vis.channels.x1}
          grow
        />
      </Align.Space>
    </Align.Space>
  );
};
