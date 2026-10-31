// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { type Select } from "@synnaxlabs/lyra/select";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Access, Channel, Log, Notation, Telem } from "@synnaxlabs/pluto";
import {
  color,
  DataType,
  type notation,
  primitive,
  type TimestampFormat,
  type TimeZone,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

const PRECISION_BOUNDS = { lower: 0, upper: 17 };

const showsNumericFields = (dt: DataType | undefined): boolean =>
  dt != null && dt.isNumeric && !dt.equals(DataType.TIMESTAMP);

const isTimestamp = (dt: DataType | undefined): boolean =>
  dt != null && dt.equals(DataType.TIMESTAMP);

interface ChannelRowProps {
  index: number;
  channel: channel.Key;
  disabled: boolean;
}

const SHADOW_TRIGGER_PROPS: Select.SingleTriggerProps = { variant: "shadow" };

const ChannelRow = ({
  index,
  channel,
  disabled,
}: ChannelRowProps): ReactElement | null => {
  const dispatch = Log.useSingleDispatch();
  const theme = Theming.use();
  const defaultColor = theme.colors.gray.l11;
  const config = Log.useChannelEntry({ channel });
  const query = channel > 0 ? { key: channel } : null;
  const { data: dataType } = Channel.useResultDataType(query);
  const { data: chName } = Channel.useResultName(query);
  const showNumeric = showsNumericFields(dataType);
  const showTimestamp = isTimestamp(dataType);

  const handleAliasChange = (alias: string) =>
    dispatch(log.setChannelAlias({ channel, alias }));
  const handleNotationChange = (notation: notation.Notation) =>
    dispatch(log.setChannelNotation({ channel, notation }));
  const handlePrecisionChange = (precision?: number) =>
    dispatch(log.setChannelPrecision({ channel, precision }));
  const handleFormatChange = (format: TimestampFormat) =>
    dispatch(log.setChannelTimestampFormat({ channel, format }));
  const handleTzChange = (tz: TimeZone) =>
    dispatch(log.setChannelTimestampTz({ channel, tz }));
  // The log stores a zero color for a channel that follows the theme.
  const handleColorChange = (c?: color.Color) =>
    dispatch(log.setChannelColor({ channel, color: c ?? color.ZERO }));
  // Clearing the row's channel removes the row.
  const handleChannelChange = (to: channel.Key | null) =>
    dispatch(
      to == null
        ? log.removeChannel({ channel })
        : log.swapChannel({ from: channel, to }),
    );
  if (config == null) return null;
  const { alias, timestamp, precision, notation } = config;
  const hasCustomColor = !color.isZero(config.color);
  return (
    <List.Item
      itemKey={channel}
      key={channel}
      index={index}
      selected={false}
      align="center"
      justify="between"
      gap="large"
      className={CSS.BE("log", "channel-row")}
      role="listitem"
      preventClick
    >
      <Flex.Box x align="center" grow>
        <Channel.SelectSingle
          value={channel}
          onChange={handleChannelChange}
          allowNone
          initialQuery={SELECT_CHANNEL_INITIAL_QUERY}
          disabled={disabled}
          triggerProps={SHADOW_TRIGGER_PROPS}
          className={CSS.BE("log", "channel-select")}
        />
        <Input.Text
          value={alias}
          onChange={handleAliasChange}
          disabled={disabled}
          placeholder={chName ?? "Alias"}
          variant="shadow"
          shrink={false}
          startContent={<Icon.Rename />}
          tooltip="Alias"
          className={CSS.BE("log", "channel-alias")}
        />
      </Flex.Box>
      <Flex.Box x align="center">
        {showNumeric && (
          <>
            <Notation.Select value={notation} onChange={handleNotationChange} />
            <Input.Numeric
              value={precision}
              onChange={handlePrecisionChange}
              placeholder="Auto"
              bounds={PRECISION_BOUNDS}
              disabled={disabled}
              shrink={false}
              variant="shadow"
              startContent={<Icon.Decimal />}
              tooltip="Precision"
              className={CSS.BE("log", "channel-precision")}
              showDragHandle={false}
            >
              <Button.Button
                variant="outlined"
                disabled={disabled || precision == null}
                onClick={() => handlePrecisionChange()}
                tooltip={
                  precision == null
                    ? "Type a number to disable auto precision"
                    : "Enable auto precision"
                }
              >
                <Icon.Auto />
              </Button.Button>
            </Input.Numeric>
          </>
        )}
        {showTimestamp && (
          <>
            <Telem.SelectTimestampFormat
              value={timestamp.format}
              onChange={handleFormatChange}
              triggerProps={SHADOW_TRIGGER_PROPS}
            />
            <Telem.SelectTimeZone
              className={CSS.BE("log", "channel-tz")}
              value={timestamp.tz}
              onChange={handleTzChange}
            />
          </>
        )}
        <Color.Swatch
          value={hasCustomColor ? config.color : undefined}
          fallback={defaultColor}
          onChange={handleColorChange}
          size="small"
        />
      </Flex.Box>
    </List.Item>
  );
};

const SELECT_CHANNEL_INITIAL_QUERY: Channel.ListQuery = {
  where: { internal: IS_DEV ? undefined : { eq: false } },
};

const ADD_CHANNEL_TRIGGER_PROPS: Select.SingleTriggerProps = {
  placeholder: "Add channel",
};

interface AddChannelRowProps {
  disabled: boolean;
}

const AddChannelRow = ({ disabled }: AddChannelRowProps): ReactElement => {
  const dispatch = Log.useSingleDispatch();
  const handleAdd = (channel: channel.Key) => dispatch(log.addChannel({ channel }));
  return (
    <Flex.Box x align="center" gap="large" className={CSS.BE("log", "channel-row")}>
      <Channel.SelectSingle
        allowNone
        onChange={handleAdd}
        initialQuery={SELECT_CHANNEL_INITIAL_QUERY}
        disabled={disabled}
        triggerProps={ADD_CHANNEL_TRIGGER_PROPS}
        className={CSS.BE("log", "channel-select")}
      />
    </Flex.Box>
  );
};

export const Channels = (): ReactElement => {
  const channelKeys = Log.useChannelKeys();
  const key = Log.useKey();
  const hasUpdatePermission = Access.useUpdateGranted(log.ontologyID(key));
  return (
    <Flex.Box
      y
      full="y"
      role="list"
      aria-label="Log channels"
      className={CSS.BE("log", "toolbar", "channels")}
    >
      {channelKeys.map((ch, i) =>
        primitive.isZero(ch) ? null : (
          <ChannelRow key={ch} index={i} channel={ch} disabled={!hasUpdatePermission} />
        ),
      )}
      <AddChannelRow disabled={!hasUpdatePermission} />
    </Flex.Box>
  );
};
