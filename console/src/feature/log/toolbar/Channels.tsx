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
  type TimestampFormat,
  type TimeZone,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

const PRECISION_BOUNDS = { lower: 0, upper: 17 };
// The first column of every row, so the channel selectors line up.
const LEAD_CLASS = CSS.BE("log", "channel-lead");
const ALIAS_MAX_LENGTH = 64;

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
  const sessionDispatch = Session.useDispatch();
  const key = Log.useKey();
  const hidden = Session.Log.useSelectHiddenChannels().includes(channel);
  const theme = Theming.use();
  const defaultColor = theme.colors.gray.l11;
  const config = Log.useChannelEntry({ channel });
  const otherChannels = Log.useChannelKeys().filter((key) => key !== channel);
  const query = { key: channel };
  const { data: dataType } = Channel.useResultDataType(query);
  const { data: chName } = Channel.useResultName(query);
  const showNumeric = showsNumericFields(dataType);
  const showTimestamp = isTimestamp(dataType);

  // Spreading counts each character as one, which the input's maxLength does not.
  const handleAliasChange = (alias: string) =>
    dispatch(
      log.setChannelAlias({
        channel,
        alias: [...alias].slice(0, ALIAS_MAX_LENGTH).join(""),
      }),
    );
  const handleNotationChange = (notation: notation.Notation) =>
    dispatch(log.setChannelNotation({ channel, notation }));
  const handlePrecisionChange = (precision?: number) =>
    dispatch(log.setChannelPrecision({ channel, precision }));
  const handleFormatChange = (format: TimestampFormat) =>
    dispatch(log.setChannelTimestampFormat({ channel, format }));
  const handleTimeZoneChange = (timeZone: TimeZone) =>
    dispatch(log.setChannelTimestampTimeZone({ channel, timeZone }));
  // The log stores a zero color for a channel that follows the theme.
  const handleColorChange = (c?: color.Color) =>
    dispatch(log.setChannelColor({ channel, color: c ?? color.ZERO }));
  const handleVisibleChange = () =>
    sessionDispatch(Session.Log.setChannelVisible({ key, channel, visible: hidden }));
  // Clearing the row's channel removes the row.
  const handleChannelChange = (to: channel.Key | null) =>
    dispatch(
      to == null
        ? log.removeChannel({ channel })
        : log.swapChannel({ from: channel, to }),
    );
  if (config == null) return null;
  const { alias, timestampConfig, precision, notation } = config;
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
        <Button.Button
          onClick={handleVisibleChange}
          size="small"
          variant="text"
          tooltip={hidden ? "Show channel" : "Hide channel"}
          aria-label={hidden ? "Show channel" : "Hide channel"}
          className={LEAD_CLASS}
        >
          {hidden ? <Icon.Hidden /> : <Icon.Visible />}
        </Button.Button>
        <Channel.SelectSingle
          value={channel}
          onChange={handleChannelChange}
          allowNone
          query={selectQueryExcluding(otherChannels)}
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
            <Notation.Select
              value={notation}
              onChange={handleNotationChange}
              disabled={disabled}
            />
            <Input.Numeric
              value={precision}
              onChange={handlePrecisionChange}
              onClear={() => handlePrecisionChange()}
              endContent={precision == null ? <Input.AutoBadge /> : undefined}
              bounds={PRECISION_BOUNDS}
              disabled={disabled}
              shrink={false}
              variant="shadow"
              startContent={<Icon.Decimal />}
              tooltip="Precision"
              className={CSS.BE("log", "channel-precision")}
              showDragHandle={false}
            >
              {precision != null && (
                <Input.AutoButton
                  disabled={disabled}
                  onClick={() => handlePrecisionChange()}
                />
              )}
            </Input.Numeric>
          </>
        )}
        {showTimestamp && (
          <>
            <Telem.SelectTimestampFormat
              value={timestampConfig.format}
              onChange={handleFormatChange}
              disabled={disabled}
              triggerProps={SHADOW_TRIGGER_PROPS}
            />
            <Telem.SelectTimeZone
              className={CSS.BE("log", "channel-tz")}
              value={timestampConfig.timeZone}
              onChange={handleTimeZoneChange}
              disabled={disabled}
            />
          </>
        )}
        <Color.Swatch
          value={hasCustomColor ? config.color : undefined}
          fallback={defaultColor}
          onChange={handleColorChange}
          disabled={disabled}
          size="small"
        />
      </Flex.Box>
    </List.Item>
  );
};

const SELECT_CHANNEL_INITIAL_QUERY: Channel.ListQuery = {
  where: { internal: IS_DEV ? undefined : { eq: false } },
};

const selectQueryExcluding = (keys: channel.Key[]): Channel.ListQuery =>
  keys.length === 0
    ? SELECT_CHANNEL_INITIAL_QUERY
    : {
        where: { ...SELECT_CHANNEL_INITIAL_QUERY.where, not: { key: { in: keys } } },
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
      <Flex.Box x align="center" grow>
        <Flex.Box x align="center" justify="center" className={LEAD_CLASS}>
          <Icon.Add aria-hidden />
        </Flex.Box>
        <Channel.SelectSingle
          allowNone
          onChange={handleAdd}
          query={SELECT_CHANNEL_INITIAL_QUERY}
          disabled={disabled}
          triggerProps={ADD_CHANNEL_TRIGGER_PROPS}
          className={CSS.BE("log", "channel-select")}
        />
      </Flex.Box>
    </Flex.Box>
  );
};

export const Channels = (): ReactElement => {
  const channelKeys = Log.useChannelKeys();
  const key = Log.useKey();
  const hasUpdatePermission = Access.useUpdateGranted(log.resourceID(key));
  return (
    <Flex.Box
      y
      full="y"
      role="list"
      aria-label="Log channels"
      className={CSS.BE("log", "toolbar", "channels")}
    >
      {channelKeys.map((ch, i) => (
        <ChannelRow key={ch} index={i} channel={ch} disabled={!hasUpdatePermission} />
      ))}
      <AddChannelRow disabled={!hasUpdatePermission} />
    </Flex.Box>
  );
};
