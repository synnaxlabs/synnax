// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/ListAndDetailsChannelItem.css";

import { type channel } from "@synnaxlabs/client";
import { Flex, type List, Select, Text, Tooltip } from "@synnaxlabs/pluto";
import { type direction, type record } from "@synnaxlabs/x";
import { cloneElement, type CSSProperties, type JSX, useMemo } from "react";

import { CSS } from "@/platform/css";
import { ChannelName, type ChannelNameProps } from "@/platform/task/ChannelName";
import { EnableDisableButton } from "@/platform/task/EnableDisableButton";
import { getChannelNameID } from "@/platform/task/getChannelNameID";
import { TareButton } from "@/platform/task/TareButton";
import {
  type CommandStatePair,
  WriteChannelNames,
} from "@/platform/task/WriteChannelNames";

export interface ListAndDetailsIconProps {
  icon: JSX.Element;
  name: string;
}

/** A read row binds to one channel, a write row to a command and state pair. */
export type Binding = channel.Key | CommandStatePair;

export interface ListAndDetailsChannelItemProps<
  K extends record.Key,
  D,
> extends List.ItemProps<K> {
  port: string | number;
  portMaxChars: number;
  icon?: ListAndDetailsIconProps;
  canTare: boolean;
  channel: channel.Key;
  stateChannel?: channel.Key;
  /** The device whose saved map binds the row, or undefined while it loads. */
  device: D | undefined;
  /** Returns what the device map binds the row to, 0 where it has nothing. */
  resolve: (device: D) => Binding;
  onTare?: (channel: channel.Key) => void;
  path: string;
  hasTareButton: boolean;
  nameDirection?: direction.Direction;
}

const toPair = (binding: Binding): CommandStatePair =>
  typeof binding === "number" ? { command: binding, state: 0 } : binding;

const toKey = (binding: Binding): channel.Key =>
  typeof binding === "number" ? binding : binding.command;

const getChannelNameProps = (
  hasIcon: boolean,
): Omit<
  ChannelNameProps<unknown>,
  "channel" | "channelPath" | "device" | "namePath" | "resolve"
> => ({
  level: "p",
  color: 9,
  weight: 450,
  className: CSS.cls(
    CSS.BE("channel-item", "name"),
    hasIcon && CSS.BEM("channel-item", "name", "with-icon"),
  ),
  overflow: "ellipsis",
});

export const ListAndDetailsChannelItem = <K extends string, D>({
  port,
  portMaxChars,
  canTare,
  onTare,
  path,
  hasTareButton,
  channel,
  icon,
  stateChannel,
  device,
  resolve,
  nameDirection = "x",
  ...rest
}: ListAndDetailsChannelItemProps<K, D>) => {
  const { itemKey } = rest;
  const hasStateChannel = stateChannel != null;
  const hasIcon = icon != null;
  const channelNameProps = getChannelNameProps(hasIcon);
  const portStyle = useMemo<CSSProperties>(
    () => ({ width: `${portMaxChars * 1.25}rem` }),
    [portMaxChars],
  );
  return (
    <Select.ListItem
      {...rest}
      justify="between"
      align="center"
      className={CSS.B("channel-item")}
    >
      <Flex.Box
        direction={nameDirection}
        gap="small"
        align={nameDirection === "x" ? "center" : "start"}
      >
        <Text.Text color={9} weight={500} style={portStyle}>
          {port}
        </Text.Text>
        {hasIcon && (
          <Tooltip.Dialog>
            {icon.name}
            {cloneElement(icon.icon, {
              className: CSS.BE("channel-item", "icon"),
            })}
          </Tooltip.Dialog>
        )}
        {hasStateChannel ? (
          <Flex.Box direction="y" gap="small">
            <WriteChannelNames
              stateNamePath={`${path}.stateChannelName`}
              cmdNamePath={`${path}.cmdChannelName`}
              cmdChannel={channel}
              cmdChannelPath={`${path}.cmdChannel`}
              stateChannel={stateChannel}
              stateChannelPath={`${path}.stateChannel`}
              device={device}
              resolve={(d: D) => toPair(resolve(d))}
              itemKey={itemKey}
            />
          </Flex.Box>
        ) : (
          <ChannelName
            {...channelNameProps}
            channel={channel}
            channelPath={`${path}.channel`}
            device={device}
            resolve={(d: D) => toKey(resolve(d))}
            namePath={`${path}.name`}
            id={getChannelNameID(itemKey)}
          />
        )}
      </Flex.Box>
      <Flex.Box pack direction="x" align="center" size="small">
        {hasTareButton && (
          <TareButton disabled={!canTare} onTare={() => onTare?.(channel)} />
        )}
        <EnableDisableButton path={`${path}.disabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};
