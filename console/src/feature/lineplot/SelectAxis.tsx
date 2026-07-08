// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Channel, Input, LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Range } from "@/platform/range";
import { Session } from "@/session";

const SEARCH_OPTIONS: channel.RetrieveOptions = {
  notDataTypes: ["string", "json", "uuid"],
  internal: false,
  virtual: false,
};

export interface YAxisChannelSelectProps extends Omit<
  Input.ItemProps,
  "onChange" | "children" | "label"
> {
  axisKey: lineplot.YAxisKey;
}

export const YAxisChannelSelect = ({
  axisKey,
  ...rest
}: YAxisChannelSelectProps): ReactElement => {
  const value = LinePlot.useSelectYAxisChannels({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const rangeKey = Session.Range.useSelectSelectedKey();
  const handleChange = useCallback(
    (channels: channel.Key[]) => dispatch(lineplot.setChannels({ axisKey, channels })),
    [dispatch, axisKey],
  );
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest}>
      <Channel.SelectMultiple
        value={value}
        initialQuery={{ ...SEARCH_OPTIONS, rangeKey }}
        onChange={handleChange}
        full="x"
        location="top"
      />
    </Input.Item>
  );
};

export interface XAxisChannelSelectProps extends Omit<
  Input.ItemProps,
  "onChange" | "children" | "label"
> {
  axisKey: lineplot.XAxisKey;
}

export const XAxisChannelSelect = ({
  axisKey,
  ...rest
}: XAxisChannelSelectProps): ReactElement => {
  const value = LinePlot.useSelectXAxisChannel({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const rangeKey = Session.Range.useSelectSelectedKey();
  const handleChange = useCallback(
    (channel: channel.Key) => dispatch(lineplot.setXChannel({ axisKey, channel })),
    [dispatch, axisKey],
  );
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest} grow>
      <Channel.SelectSingle
        value={value}
        onChange={handleChange}
        allowNone
        initialQuery={{ ...SEARCH_OPTIONS, rangeKey }}
        location="top"
      />
    </Input.Item>
  );
};

export interface XAxisRangeSelectProps extends Omit<
  Range.SelectMultipleInputItemProps,
  "value" | "onChange"
> {
  axisKey: lineplot.XAxisKey;
}

export const XAxisRangeSelect = ({
  axisKey,
  ...rest
}: XAxisRangeSelectProps): ReactElement => {
  const value = LinePlot.useSelectXAxisRanges({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = useCallback(
    (ranges: string[]) => dispatch(lineplot.setRanges({ axisKey, ranges })),
    [dispatch, axisKey],
  );
  return (
    <Range.SelectMultipleInputItem value={value} onChange={handleChange} {...rest} />
  );
};
