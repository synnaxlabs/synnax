// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Flex, LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import {
  SelectAxisInputItem,
  SelectMultipleAxesInputItem,
} from "@/lineplot/SelectAxis";
import { Range } from "@/range";

const SELECT_PROPS = { location: "top" } as const;

export interface DataProps {
  layoutKey: string;
}

const diffChannels = (
  current: readonly channel.Key[],
  next: readonly channel.Key[],
  axisKey: lineplot.YAxisKey,
): lineplot.Action[] => {
  const currentSet = new Set(current);
  const nextSet = new Set(next);
  const actions: lineplot.Action[] = [];
  for (const channel of current)
    if (!nextSet.has(channel))
      actions.push(lineplot.removeChannel({ axisKey, channel }));
  for (const channel of next)
    if (!currentSet.has(channel))
      actions.push(lineplot.addChannel({ axisKey, channel }));
  return actions;
};

const diffRanges = (
  current: readonly string[],
  next: readonly string[],
  axisKey: lineplot.XAxisKey,
): lineplot.Action[] => {
  const currentSet = new Set(current);
  const nextSet = new Set(next);
  const actions: lineplot.Action[] = [];
  for (const range of current)
    if (!nextSet.has(range)) actions.push(lineplot.removeRange({ axisKey, range }));
  for (const range of next)
    if (!currentSet.has(range)) actions.push(lineplot.addRange({ axisKey, range }));
  return actions;
};

export const Data = ({ layoutKey }: DataProps): ReactElement => {
  const channels = LinePlot.useSelectChannels({ key: layoutKey });
  const ranges = LinePlot.useSelectRanges({ key: layoutKey });
  const { dispatch } = LinePlot.useDispatch();

  const handleYChannelSelect = useCallback(
    (key: lineplot.AxisKey, value: readonly channel.Key[]): void => {
      const axis = key as lineplot.YAxisKey;
      dispatch({ key: layoutKey, actions: diffChannels(channels[axis], value, axis) });
    },
    [dispatch, layoutKey, channels],
  );

  const handleXChannelSelect = useCallback(
    (key: lineplot.AxisKey, value: channel.Key): void => {
      const axis = key as lineplot.XAxisKey;
      dispatch({
        key: layoutKey,
        actions: [lineplot.setXChannel({ axisKey: axis, channel: value })],
      });
    },
    [dispatch, layoutKey],
  );

  const handleRangeSelect = useCallback(
    (key: lineplot.XAxisKey, value: string[]): void => {
      dispatch({ key: layoutKey, actions: diffRanges(ranges[key], value, key) });
    },
    [dispatch, layoutKey, ranges],
  );

  return (
    <Flex.Box className={CSS.BE("line-plot", "toolbar", "data")} full="x">
      <SelectMultipleAxesInputItem
        axis="y1"
        onChange={handleYChannelSelect}
        value={channels.y1}
        align="center"
        grow
        selectProps={SELECT_PROPS}
      />
      <SelectMultipleAxesInputItem
        axis="y2"
        onChange={handleYChannelSelect}
        value={channels.y2}
        grow
        selectProps={SELECT_PROPS}
      />
      <Flex.Box x grow wrap>
        <Range.SelectMultipleInputItem
          onChange={(v) => handleRangeSelect("x1", v)}
          value={ranges.x1}
          grow
        />
        <SelectAxisInputItem
          axis="x1"
          className={CSS.BE("line-plot", "toolbar", "data-x")}
          onChange={handleXChannelSelect}
          value={channels.x1}
          selectProps={SELECT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
  );
};
