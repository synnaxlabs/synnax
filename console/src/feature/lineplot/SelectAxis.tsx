// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, lineplot } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

const SEARCH_QUERY: Channel.ListQuery = {
  where: {
    not: { dataType: { in: [DataType.STRING, DataType.JSON, DataType.UUID] } },
    virtual: false,
  },
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
  const handleChange = (channels: channel.Key[]) =>
    dispatch(lineplot.setChannels({ axisKey, channels }));
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest}>
      <Channel.SelectMultiple
        value={value}
        query={SEARCH_QUERY}
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

const BADGE_CLASS = CSS.BE("line-plot", "x-time-badge");

// Key 0 means no X channel: each Y channel plots against its own index channel.
const TIME_OPTION = (
  <Flex.Box className={CSS.BE("line-plot", "x-time")} full="x">
    <Select.Item<channel.Key>
      itemKey={0}
      textValue="Time"
      label={
        <>
          Time
          <Input.AutoBadge className={BADGE_CLASS} />
        </>
      }
    >
      <Icon.Time />
      <Text.Text>
        Time
        <Input.AutoBadge className={BADGE_CLASS} />
      </Text.Text>
      <Select.ItemDescription>
        Automatically uses each channel's time. Best for most needs.
      </Select.ItemDescription>
    </Select.Item>
  </Flex.Box>
);

export const XAxisChannelSelect = ({
  axisKey,
  ...rest
}: XAxisChannelSelectProps): ReactElement => {
  const value = LinePlot.useXAxisChannel({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const handleChange = (channel: channel.Key | null) =>
    dispatch(lineplot.setXChannel({ axisKey, channel: channel ?? 0 }));
  return (
    <Input.Item x label={LinePlot.axisLabel(axisKey)} {...rest} grow>
      <Channel.SelectSingle
        value={value}
        onChange={handleChange}
        allowNone
        query={SEARCH_QUERY}
        variant="floating"
        location="top"
        icon={<Icon.Time />}
        footer={TIME_OPTION}
      />
    </Input.Item>
  );
};
