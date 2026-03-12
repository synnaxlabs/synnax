// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log } from "@synnaxlabs/client";
import { Access, Channel, Color, Input, List } from "@synnaxlabs/pluto";
import { color, DataType, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import { type ChannelConfig, setChannelConfig, ZERO_CHANNEL_CONFIG } from "@/log/slice";

interface ChannelRowProps {
  itemKey: channel.Key;
  index: number;
  config: ChannelConfig;
  onChange: (channelKey: channel.Key, config: Partial<ChannelConfig>) => void;
  disabled: boolean;
}

const PRECISION_BOUNDS = { lower: -1, upper: 17 };

const ChannelRow = ({
  itemKey: channelKey,
  index,
  config,
  onChange,
  disabled,
}: ChannelRowProps): ReactElement => {
  const { data } = Channel.useRetrieve({ key: channelKey });
  const name = data?.name ?? String(channelKey);
  const isFloat =
    data?.dataType.equals(DataType.FLOAT32) === true ||
    data?.dataType.equals(DataType.FLOAT64) === true;

  const handleColorChange = (c: color.Color): void =>
    onChange(channelKey, { color: color.hex(c) });

  const handlePrecisionChange = (v: number): void =>
    onChange(channelKey, { precision: v });

  return (
    <List.Item
      key={channelKey}
      itemKey={channelKey}
      index={index}
      align="center"
      gap="medium"
    >
      <Input.Text
        value={name}
        onChange={() => {}}
        readOnly
        grow
        variant="shadow"
        preventClick
      />
      <Input.Numeric
        value={config.precision}
        onChange={handlePrecisionChange}
        resetValue={-1}
        bounds={PRECISION_BOUNDS}
        disabled={disabled || !isFloat}
        shrink={false}
        variant="shadow"
        tooltip="Decimal places (-1 = no rounding)"
      />
      <Color.Swatch
        value={config.color !== "" ? config.color : color.ZERO}
        onChange={handleColorChange}
        size="small"
        disabled={disabled}
      />
    </List.Item>
  );
};

export interface TextProps {
  layoutKey: string;
}

const EMPTY_CONTENT = <EmptyAction message="No channels configured." />;

export const Text = ({ layoutKey }: TextProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const state = useSelectOptional(layoutKey);
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(layoutKey));
  const handleChange = (chKey: channel.Key, cfg: Partial<ChannelConfig>): void => {
    dispatch(setChannelConfig({ key: layoutKey, channelKey: chKey, config: cfg }));
  };
  if (state == null) return null;
  const activeChannels = state.channels.filter((ch) => !primitive.isZero(ch));
  return (
    <List.Frame data={activeChannels}>
      <List.Items<channel.Key>
        full="y"
        className={CSS.BE("log", "toolbar", "text")}
        emptyContent={EMPTY_CONTENT}
      >
        {({ key, index }) => (
          <ChannelRow
            key={key}
            itemKey={key}
            index={index}
            config={state.channelConfigs[String(key)] ?? ZERO_CHANNEL_CONFIG}
            onChange={handleChange}
            disabled={!hasEditPermission}
          />
        )}
      </List.Items>
    </List.Frame>
  );
};
