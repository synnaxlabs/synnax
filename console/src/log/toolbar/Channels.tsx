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

import { CSS } from "@/css";

const PRECISION_BOUNDS = { lower: -1, upper: 18 };

const showsNumericFields = (dt: DataType | undefined): boolean =>
  dt != null && dt.isNumeric && !dt.equals(DataType.TIMESTAMP);

const isTimestamp = (dt: DataType | undefined): boolean =>
  dt != null && dt.equals(DataType.TIMESTAMP);

interface ChannelRowProps {
  index: number;
  channelKey: channel.Key;
  ch: channel.Channel | undefined;
  config: log.ChannelEntry;
  onChange: (prevKey: channel.Key, nextKey: channel.Key) => void;
  onConfigChange: (
    channelKey: channel.Key,
    config: Partial<Omit<log.ChannelEntry, "channel">>,
  ) => void;
  onRemove: (index: number) => void;
  disabled: boolean;
}

const ChannelRow = ({
  index,
  channelKey,
  ch,
  config,
  onChange,
  onConfigChange,
  onRemove,
  disabled,
}: ChannelRowProps): ReactElement => {
  const theme = Theming.use();
  const defaultColor = theme.colors.gray.l11;
  const hasCustomColor = !color.isZero(config.color);
  const showNumeric = showsNumericFields(ch?.dataType);
  const showTimestamp = isTimestamp(ch?.dataType);

  return (
    <List.Item
      itemKey={channelKey}
      key={channelKey}
      index={index}
      selected={false}
      align="center"
      justify="between"
      gap="large"
      className={CSS.BE("log", "channel-row")}
    >
      <Flex.Box x align="center" grow>
        <Channel.SelectSingle
          value={channelKey}
          onChange={(v: channel.Key) => onChange(channelKey, v)}
          initialQuery={{ internal: IS_DEV ? undefined : false }}
          disabled={disabled}
          className={CSS.BE("log", "channel-select")}
        />
        <Input.Text
          value={config.alias ?? ""}
          onChange={(v) => onConfigChange(channelKey, { alias: v })}
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
            <Notation.Select
              value={config.notation ?? "standard"}
              onChange={(v: notation.Notation) =>
                onConfigChange(channelKey, { notation: v })
              }
            />
            <Input.Numeric
              value={config.precision}
              onChange={(v) => onConfigChange(channelKey, { precision: v })}
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
                disabled={disabled || config.precision === -1}
                onClick={() => onConfigChange(channelKey, { precision: -1 })}
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
              value={config.timestamp.format}
              onChange={(f: TimestampFormat) =>
                onConfigChange(channelKey, {
                  timestamp: { ...config.timestamp, format: f },
                })
              }
            />
            <Telem.SelectTimeZone
              className={CSS.BE("log", "channel-tz")}
              value={config.timestamp.tz}
              onChange={(v: TimeZone) =>
                onConfigChange(channelKey, {
                  timestamp: { ...config.timestamp, tz: v },
                })
              }
            />
          </>
        )}
        <Color.Swatch
          value={hasCustomColor ? config.color : defaultColor}
          onChange={(c) => onConfigChange(channelKey, { color: c })}
          onDelete={
            hasCustomColor
              ? () => onConfigChange(channelKey, { color: color.ZERO })
              : undefined
          }
          size="small"
        />
        <Button.Button
          onClick={() => onRemove(index)}
          size="small"
          variant="text"
          ghost
          tooltip="Remove channel"
          contrast={0}
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </List.Item>
  );
};

interface AddChannelRowProps {
  onAdd: (channelKey: channel.Key) => void;
  disabled: boolean;
}

const AddChannelRow = ({ onAdd, disabled }: AddChannelRowProps): ReactElement => (
  <Flex.Box x align="center" gap="large" className={CSS.BE("log", "channel-row")}>
    <Channel.SelectSingle
      value={0}
      onChange={onAdd}
      initialQuery={{ internal: IS_DEV ? undefined : false }}
      disabled={disabled}
      triggerProps={{ placeholder: "Add a channel..." }}
      className={CSS.BE("log", "channel-select")}
    />
  </Flex.Box>
);

export interface ChannelsProps {
  layoutKey: string;
}

export const Channels = ({ layoutKey }: ChannelsProps): ReactElement => {
  const { dispatch } = Log.useDispatch();
  const channels = Log.useSelectChannels({ key: layoutKey });
  const hasUpdatePermission = Access.useUpdateGranted(log.ontologyID(layoutKey));

  const apply = useCallback(
    (action: log.Action) => dispatch({ key: layoutKey, actions: [action] }),
    [dispatch, layoutKey],
  );

  const channelKeys = useMemo(
    () => channels.map((c) => c.channel).filter((k) => !primitive.isZero(k)),
    [channels],
  );
  const { data: retrieved } = Channel.useRetrieveMultiple({ keys: channelKeys });

  const handleChannelChange = useCallback(
    (prevKey: channel.Key, nextKey: channel.Key) =>
      apply(log.swapChannel({ from: prevKey, to: nextKey })),
    [apply],
  );

  const handleConfigChange = useCallback(
    (
      channelKey: channel.Key,
      config: Partial<Omit<log.ChannelEntry, "channel">>,
    ): void => {
      const current = channels.find((e) => e.channel === channelKey);
      if (current == null) return;
      apply(log.setChannelEntry({ entry: { ...current, ...config } }));
    },
    [apply, channels],
  );

  const handleRemove = useCallback(
    (index: number) => {
      const entry = channels[index];
      if (entry != null) apply(log.removeChannel({ channel: entry.channel }));
    },
    [apply, channels],
  );

  const handleAdd = useCallback(
    (channelKey: channel.Key) => apply(log.addChannel({ channel: channelKey })),
    [apply],
  );

  return (
    <Flex.Box y full="y" className={CSS.BE("log", "toolbar", "channels")}>
      {channels.map((entry, i) =>
        primitive.isZero(entry.channel) ? null : (
          <ChannelRow
            key={`${entry.channel}-${i}`}
            index={i}
            channelKey={entry.channel}
            ch={retrieved?.find((c) => c.key === entry.channel)}
            config={entry}
            onChange={handleChannelChange}
            onConfigChange={handleConfigChange}
            onRemove={handleRemove}
            disabled={!hasUpdatePermission}
          />
        ),
      )}
      <AddChannelRow onAdd={handleAdd} disabled={!hasUpdatePermission} />
    </Flex.Box>
  );
};
