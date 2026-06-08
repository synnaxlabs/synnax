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

export const Data = ({ layoutKey }: DataProps): ReactElement => {
  const channels = LinePlot.useSelectChannels({ key: layoutKey });
  const ranges = LinePlot.useSelectRanges({ key: layoutKey });
  const { dispatch } = LinePlot.useDispatch();

  const handleYChannelSelect = useCallback(
    (key: lineplot.AxisKey, value: readonly channel.Key[]): void => {
      const axisKey = key as lineplot.YAxisKey;
      dispatch({
        key: layoutKey,
        actions: [lineplot.setChannels({ axisKey, channels: [...value] })],
      });
    },
    [dispatch, layoutKey],
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
    (axisKey: lineplot.XAxisKey, value: string[]): void => {
      dispatch({
        key: layoutKey,
        actions: [lineplot.setRanges({ axisKey, ranges: value })],
      });
    },
    [dispatch, layoutKey],
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
