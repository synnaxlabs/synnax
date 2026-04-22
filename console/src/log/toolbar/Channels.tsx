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
  type Log as PLog,
  Notation,
  Select,
  Theming,
} from "@synnaxlabs/pluto";
import { color, DataType, type notation, primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/css";
import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import {
  addChannel,
  type ChannelConfig,
  removeChannelByIndex,
  setChannelAtIndex,
  setChannelConfig,
} from "@/log/slice";

const PRECISION_BOUNDS = { lower: -1, upper: 18 };

const showsNumericFields = (dt: DataType | undefined): boolean =>
  dt != null && dt.isNumeric && !dt.equals(DataType.TIMESTAMP);

const isTimestamp = (dt: DataType | undefined): boolean =>
  dt != null && dt.equals(DataType.TIMESTAMP);

const TIMESTAMP_FORMATS = ["preciseTime", "preciseDate", "ISO"] as const;
const TIMESTAMP_TZS = ["UTC", "local"] as const;

interface TimestampFormatSelectProps extends Omit<
  Select.ButtonsProps<PLog.TimestampFormat>,
  "keys"
> {}

const TimestampFormatSelect = (props: TimestampFormatSelectProps): ReactElement => (
  <Select.Buttons {...props} keys={TIMESTAMP_FORMATS}>
    <Select.Button itemKey="preciseTime" tooltip="Time">
      <Icon.Time />
    </Select.Button>
    <Select.Button itemKey="preciseDate" tooltip="Date and time">
      <Icon.Calendar />
      +
      <Icon.Time />
    </Select.Button>
    <Select.Button itemKey="ISO" tooltip="ISO 8601">
      ISO
    </Select.Button>
  </Select.Buttons>
);

interface TimestampTZSelectProps extends Omit<
  Select.ButtonsProps<PLog.TimestampTZ>,
  "keys"
> {}

const TimestampTZSelect = (props: TimestampTZSelectProps): ReactElement => (
  <Select.Buttons {...props} keys={TIMESTAMP_TZS}>
    <Select.Button itemKey="UTC" tooltip="UTC">
      UTC
    </Select.Button>
    <Select.Button itemKey="local" tooltip="Local timezone">
      Local
    </Select.Button>
  </Select.Buttons>
);

interface ChannelRowProps {
  index: number;
  channelKey: channel.Key;
  ch: channel.Channel | undefined;
  config: ChannelConfig;
  onChange: (index: number, channelKey: channel.Key) => void;
  onConfigChange: (channelKey: channel.Key, config: Partial<ChannelConfig>) => void;
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
  const hasCustomColor = config.color !== "";
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
          onChange={(v: channel.Key) => onChange(index, v)}
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
            <TimestampFormatSelect
              value={config.timestamp.format}
              onChange={(v: PLog.TimestampFormat) =>
                onConfigChange(channelKey, {
                  timestamp: { ...config.timestamp, format: v },
                })
              }
            />
            <TimestampTZSelect
              value={config.timestamp.tz}
              onChange={(v: PLog.TimestampTZ) =>
                onConfigChange(channelKey, {
                  timestamp: { ...config.timestamp, tz: v },
                })
              }
            />
          </>
        )}
        <Color.Swatch
          value={hasCustomColor ? config.color : defaultColor}
          onChange={(c) => onConfigChange(channelKey, { color: color.hex(c) })}
          onDelete={
            hasCustomColor ? () => onConfigChange(channelKey, { color: "" }) : undefined
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

export const Channels = ({ layoutKey }: ChannelsProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const state = useSelectOptional(layoutKey);
  const hasUpdatePermission = Access.useUpdateGranted(log.ontologyID(layoutKey));

  const channelKeys = useMemo(
    () =>
      state?.channels.map((c) => c.channel).filter((k) => !primitive.isZero(k)) ?? [],
    [state?.channels],
  );
  const { data: channels } = Channel.useRetrieveMultiple({ keys: channelKeys });

  const handleChannelChange = useCallback(
    (index: number, channelKey: channel.Key) =>
      dispatch(setChannelAtIndex({ key: layoutKey, index, channelKey })),
    [dispatch, layoutKey],
  );

  const handleConfigChange = useCallback(
    (channelKey: channel.Key, config: Partial<ChannelConfig>): void => {
      dispatch(setChannelConfig({ key: layoutKey, channelKey, config }));
    },
    [dispatch, layoutKey],
  );

  const handleRemove = useCallback(
    (index: number) => dispatch(removeChannelByIndex({ key: layoutKey, index })),
    [dispatch, layoutKey],
  );

  const handleAdd = useCallback(
    (channelKey: channel.Key) => dispatch(addChannel({ key: layoutKey, channelKey })),
    [dispatch, layoutKey],
  );

  if (state == null) return null;

  return (
    <Flex.Box y full="y" className={CSS.BE("log", "toolbar", "channels")}>
      {state.channels.map((entry, i) =>
        primitive.isZero(entry.channel) ? null : (
          <ChannelRow
            key={`${entry.channel}-${i}`}
            index={i}
            channelKey={entry.channel}
            ch={channels?.find((c) => c.key === entry.channel)}
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
