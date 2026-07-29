// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Channel,
  Color,
  Flex,
  Icon,
  Input,
  List,
  Log,
  Notation,
  type Select,
  Telem,
  Theming,
} from "@synnaxlabs/pluto";
import {
  color,
  DataType,
  type notation,
  primitive,
  type TimestampFormat,
  type TimeZone,
} from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/platform/css";

const PRECISION_BOUNDS = { lower: -1, upper: 17 };

const showsNumericFields = (dt: DataType | undefined): boolean =>
  dt != null && dt.isNumeric && !dt.equals(DataType.TIMESTAMP);

const isTimestamp = (dt: DataType | undefined): boolean =>
  dt != null && dt.equals(DataType.TIMESTAMP);

interface ChannelRowProps {
  index: number;
  ch: channel.Channel | undefined;
  config: log.ChannelEntry;
  disabled: boolean;
}

const ChannelRow = ({ index, ch, config, disabled }: ChannelRowProps): ReactElement => {
  const dispatch = Log.useSingleDispatch();
  const theme = Theming.use();
  const defaultColor = theme.colors.gray.l11;
  const hasCustomColor = !color.isZero(config.color);
  const showNumeric = showsNumericFields(ch?.dataType);
  const showTimestamp = isTimestamp(ch?.dataType);
  const { channel, alias, timestamp, precision, notation } = config;

  const handleAliasChange = useCallback(
    (alias: string) => dispatch(log.setChannelAlias({ channel, alias })),
    [channel, dispatch],
  );
  const handleNotationChange = useCallback(
    (notation: notation.Notation) =>
      dispatch(log.setChannelNotation({ channel, notation })),
    [channel, dispatch],
  );
  const handlePrecisionChange = useCallback(
    (precision: number) => dispatch(log.setChannelPrecision({ channel, precision })),
    [channel, dispatch],
  );
  const handleFormatChange = useCallback(
    (format: TimestampFormat) =>
      dispatch(log.setChannelTimestampFormat({ channel, format })),
    [channel, dispatch],
  );
  const handleTzChange = useCallback(
    (tz: TimeZone) => dispatch(log.setChannelTimestampTz({ channel, tz })),
    [channel, dispatch],
  );
  const handleColorChange = useCallback(
    (color: color.Color) => dispatch(log.setChannelColor({ channel, color })),
    [channel, dispatch],
  );
  const handleChannelChange = useCallback(
    (to: channel.Key) => dispatch(log.swapChannel({ from: channel, to })),
    [dispatch],
  );

  const handleRemove = useCallback(
    () => dispatch(log.removeChannel({ channel })),
    [dispatch],
  );
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
    >
      <Flex.Box x align="center" grow>
        <Channel.SelectSingle
          value={channel}
          onChange={handleChannelChange}
          initialQuery={SELECT_CHANNEL_INITIAL_QUERY}
          disabled={disabled}
          className={CSS.BE("log", "channel-select")}
        />
        <Input.Text
          value={alias}
          onChange={handleAliasChange}
          disabled={disabled}
          placeholder={ch?.name ?? "Alias"}
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
              resetValue={-1}
              emptyValue={-1}
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
                disabled={disabled || precision === -1}
                onClick={() => handlePrecisionChange(-1)}
                tooltip={
                  config.precision === -1
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
            />
            <Telem.SelectTimeZone
              className={CSS.BE("log", "channel-tz")}
              value={timestamp.tz}
              onChange={handleTzChange}
            />
          </>
        )}
        <Color.Swatch
          value={hasCustomColor ? config.color : defaultColor}
          onChange={handleColorChange}
          onDelete={hasCustomColor ? () => handleColorChange(color.ZERO) : undefined}
          size="small"
        />
        <Button.Button
          onClick={handleRemove}
          size="small"
          variant="text"
          ghost
          tooltip="Remove channel"
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </List.Item>
  );
};

const SELECT_CHANNEL_INITIAL_QUERY: Channel.ListQuery = {
  internal: IS_DEV ? undefined : false,
};

const ADD_CHANNEL_TRIGGER_PROPS: Select.SingleTriggerProps = {
  placeholder: "Add a channel...",
};

interface AddChannelRowProps {
  disabled: boolean;
}

const AddChannelRow = ({ disabled }: AddChannelRowProps): ReactElement => {
  const dispatch = Log.useSingleDispatch();
  const handleAdd = useCallback(
    (channel: channel.Key) => dispatch(log.addChannel({ channel })),
    [dispatch],
  );
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
  const entries = Log.useSelectChannels();
  const key = Log.useKey();
  const hasUpdatePermission = Access.useUpdateGranted(log.ontologyID(key));
  const keys = useMemo(
    () => entries.map((c) => c.channel).filter((k) => !primitive.isZero(k)),
    [entries],
  );
  const { data: channels } = Channel.useRetrieveMultiple({ keys });
  return (
    <Flex.Box y full="y" className={CSS.BE("log", "toolbar", "channels")}>
      {entries.map((entry, i) =>
        primitive.isZero(entry.channel) ? null : (
          <ChannelRow
            key={entry.channel}
            index={i}
            ch={channels?.find((c) => c.key === entry.channel)}
            config={entry}
            disabled={!hasUpdatePermission}
          />
        ),
      )}
      <AddChannelRow disabled={!hasUpdatePermission} />
    </Flex.Box>
  );
};
