// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Flex, LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { type CSSProperties, type ReactElement, useCallback } from "react";

import {
  SelectAxisInputItem,
  SelectMultipleAxesInputItem,
} from "@/lineplot/SelectAxis";
import { useAssignedDispatch } from "@/lineplot/useAssignedDispatch";
import { Range } from "@/range";

const SELECT_PROPS = { location: "top" } as const;

export interface DataProps {
  layoutKey: string;
}

const SELECT_X_STYLE: CSSProperties = { maxWidth: 400, width: "100%" };

const diffChannels = (
  current: readonly channel.Key[],
  next: readonly channel.Key[],
  axisKey: lineplot.YAxisKey,
): lineplot.Action[] => {
  const currentSet = new Set(current);
  const nextSet = new Set(next);
  const actions: lineplot.Action[] = [];
  for (const c of current)
    if (!nextSet.has(c)) actions.push(lineplot.removeChannel({ axisKey, channel: c }));
  for (const c of next)
    if (!currentSet.has(c)) actions.push(lineplot.addChannel({ axisKey, channel: c }));
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
  for (const r of current)
    if (!nextSet.has(r)) actions.push(lineplot.removeRange({ axisKey, range: r }));
  for (const r of next)
    if (!currentSet.has(r)) actions.push(lineplot.addRange({ axisKey, range: r }));
  return actions;
};

export const Data = ({ layoutKey }: DataProps): ReactElement => {
  const channels = PLinePlot.useSelectChannels({ key: layoutKey });
  const ranges = PLinePlot.useSelectRanges({ key: layoutKey });
  const dispatch = useAssignedDispatch(layoutKey);

  const handleYChannelSelect = useCallback(
    (key: lineplot.AxisKey, value: readonly channel.Key[]): void => {
      const axis = key as lineplot.YAxisKey;
      dispatch(diffChannels(channels[axis], value, axis));
    },
    [dispatch, channels],
  );

  const handleXChannelSelect = useCallback(
    (key: lineplot.AxisKey, value: channel.Key): void => {
      const axis = key as lineplot.XAxisKey;
      dispatch([lineplot.setXChannel({ axisKey: axis, channel: value })]);
    },
    [dispatch],
  );

  const handleRangeSelect = useCallback(
    (key: lineplot.XAxisKey, value: string[]): void => {
      dispatch(diffRanges(ranges[key], value, key));
    },
    [dispatch, ranges],
  );

  return (
    <Flex.Box style={{ padding: "2rem" }} full="x">
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
          style={SELECT_X_STYLE}
          onChange={handleXChannelSelect}
          value={channels.x1}
          selectProps={SELECT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
  );
};
