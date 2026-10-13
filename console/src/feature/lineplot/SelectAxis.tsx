// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Channel, LinePlot } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CustomRangeInput } from "@/feature/lineplot/CustomRangeInput";
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
  const value = LinePlot.useYAxisChannels({ axisKey });
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
  const value = LinePlot.useXAxisChannel({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const rangeKey = Session.Range.useSelectSelectedKey();
  const label = LinePlot.axisLabel(axisKey);
  const handleChange = useCallback(
    (channel: channel.Key | null) =>
      dispatch(lineplot.setXChannel({ axisKey, channel: channel ?? 0 })),
    [dispatch, axisKey],
  );
  return (
    <Input.Item x label={label} {...rest} grow>
      <Flex.Box x pack grow>
        <Channel.SelectSingle
          value={value}
          onChange={handleChange}
          allowNone
          initialQuery={{ ...SEARCH_OPTIONS, rangeKey }}
          location="top"
          grow
        />
        {primitive.isNonZero(value) && (
          <Button.Button
            variant="outlined"
            onClick={() => handleChange(null)}
            tooltip={`Clear ${label} channel`}
            aria-label={`Clear ${label} channel`}
          >
            <Icon.Close />
          </Button.Button>
        )}
      </Flex.Box>
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
  const value = LinePlot.useXAxisRanges({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = useCallback(
    (ranges: string[]) => dispatch(lineplot.setRanges({ axisKey, ranges })),
    [dispatch, axisKey],
  );
  return (
    <Flex.Box x grow>
      <Range.SelectMultipleInputItem value={value} onChange={handleChange} {...rest} />
      {value.includes(Range.CUSTOM_KEY) && <CustomRangeInput />}
    </Flex.Box>
  );
};
