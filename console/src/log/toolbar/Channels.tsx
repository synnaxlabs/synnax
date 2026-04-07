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
  Notation,
  Theming,
} from "@synnaxlabs/pluto";
import { color, type notation, primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

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

interface ChannelRowProps {
  index: number;
  channelKey: channel.Key;
  config: ChannelConfig;
  onChange: (index: number, channelKey: channel.Key) => void;
  onConfigChange: (channelKey: channel.Key, config: Partial<ChannelConfig>) => void;
  onRemove: (index: number) => void;
  disabled: boolean;
}

const ChannelRow = ({
  index,
  channelKey,
  config,
  onChange,
  onConfigChange,
  onRemove,
  disabled,
}: ChannelRowProps): ReactElement => {
  const { data } = Channel.useRetrieve({ key: channelKey });
  const isNumeric = data?.dataType.isNumeric === true;
  const theme = Theming.use();
  const defaultColor = theme.colors.gray.l11;
  const hasCustomColor = config.color !== "";

  return (
    <Flex.Box x align="center" gap="large" className={CSS.BE("log", "channel-row")}>
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
        placeholder={data?.name ?? "Alias"}
        variant="shadow"
        shrink={false}
        className={CSS.BE("log", "channel-alias")}
      />
      <Notation.Select
        value={config.notation ?? "standard"}
        onChange={(v: notation.Notation) => onConfigChange(channelKey, { notation: v })}
      />
      <Input.Numeric
        value={config.precision}
        onChange={(v) => onConfigChange(channelKey, { precision: v })}
        resetValue={-1}
        bounds={PRECISION_BOUNDS}
        disabled={disabled || !isNumeric}
        shrink={false}
        variant="shadow"
        tooltip="Precision (-1 = no rounding)"
        className={CSS.BE("log", "channel-precision")}
      />
      <Color.Swatch
        value={hasCustomColor ? config.color : defaultColor}
        onChange={(c) => onConfigChange(channelKey, { color: color.hex(c) })}
        onDelete={
          hasCustomColor ? () => onConfigChange(channelKey, { color: "" }) : undefined
        }
        size="small"
        disabled={disabled}
      />
      <Button.Button
        onClick={() => onRemove(index)}
        disabled={disabled}
        size="small"
        variant="text"
        ghost
        tooltip="Remove channel"
      >
        <Icon.Close />
      </Button.Button>
    </Flex.Box>
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
    <Input.Text
      value=""
      onChange={() => {}}
      disabled
      placeholder="Alias"
      variant="shadow"
      shrink={false}
      className={CSS.BE("log", "channel-alias")}
    />
    <Notation.Select value={undefined} onChange={() => {}} allowNone />
    <Input.Numeric
      value={-1}
      onChange={() => {}}
      resetValue={-1}
      bounds={PRECISION_BOUNDS}
      disabled
      shrink={false}
      variant="shadow"
      tooltip="Precision (-1 = no rounding)"
      className={CSS.BE("log", "channel-precision")}
    />
    <Color.Swatch value={color.ZERO} onChange={() => {}} size="small" disabled />
    <Button.Button size="small" variant="text" ghost disabled>
      <Icon.Close />
    </Button.Button>
  </Flex.Box>
);

export interface ChannelsProps {
  layoutKey: string;
}

export const Channels = ({ layoutKey }: ChannelsProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const state = useSelectOptional(layoutKey);
  const hasUpdatePermission = Access.useUpdateGranted(log.ontologyID(layoutKey));

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
    <Flex.Box
      y
      full="y"
      style={{ overflow: "auto" }}
      className={CSS.BE("log", "toolbar", "channels")}
    >
      {state.channels.map((entry, i) =>
        primitive.isZero(entry.channel) ? null : (
          <ChannelRow
            key={`${entry.channel}-${i}`}
            index={i}
            channelKey={entry.channel}
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
