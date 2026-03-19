// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Flex, Input } from "@synnaxlabs/pluto";
import { type CSSProperties, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { type AxisKey, type XAxisKey, type YAxisKey } from "@/lineplot/axis";
import {
  SelectAxisInputItem,
  SelectMultipleAxesInputItem,
} from "@/lineplot/SelectAxis";
import { useSelect } from "@/lineplot/selectors";
import { setAlign, setRanges, setXChannel, setYChannels } from "@/lineplot/slice";
import { Range } from "@/range";

const SELECT_PROPS = { location: "top" } as const;

export interface DataProps {
  layoutKey: string;
}

const SELECT_X_STYLE: CSSProperties = {
  maxWidth: 400,
  width: "100%",
};

export const Data = ({ layoutKey }: DataProps): ReactElement => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();

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

  const handleAlignChange = useCallback(
    (value: boolean): void => {
      dispatch(setAlign({ key: layoutKey, align: value }));
    },
    [dispatch, layoutKey],
  );

  return (
    <Flex.Box style={{ padding: "2rem" }} full="x">
      <SelectMultipleAxesInputItem
        axis="y1"
        onChange={handleYChannelSelect}
        value={vis.channels.y1}
        align="center"
        grow
        selectProps={SELECT_PROPS}
      />
      <SelectMultipleAxesInputItem
        axis="y2"
        onChange={handleYChannelSelect}
        value={vis.channels.y2}
        grow
        selectProps={SELECT_PROPS}
      />
      <Flex.Box x grow wrap>
        <Range.SelectMultipleInputItem
          onChange={(v) => handleRangeSelect("x1", v)}
          value={vis.ranges.x1}
          grow
        />
        <Input.Item label="Align Ranges" padHelpText={false} x>
          <Input.Switch value={vis.align} onChange={handleAlignChange} />
        </Input.Item>
        <SelectAxisInputItem
          axis="x1"
          style={SELECT_X_STYLE}
          onChange={handleXChannelSelect}
          value={vis.channels.x1}
          selectProps={SELECT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
  );
};
