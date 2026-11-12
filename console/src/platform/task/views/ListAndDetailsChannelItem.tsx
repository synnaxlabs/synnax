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
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { type List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { deep, type direction, type record } from "@synnaxlabs/x";
import { cloneElement, type CSSProperties, type JSX, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { ChannelName, type ChannelNameProps } from "@/platform/task/ChannelName";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { getChannelNameID } from "@/platform/task/getChannelNameID";
import { TareButton } from "@/platform/task/TareButton";
import { type Channel } from "@/platform/task/types";
import { WriteChannelNames } from "@/platform/task/WriteChannelNames";

/** A read channel carries a name. A write channel carries two names. */
interface ItemChannel extends Channel {
  name?: string;
  cmdChannelName?: string;
  stateChannelName?: string;
}

export interface ListAndDetailsIconProps {
  icon: JSX.Element;
  name: string;
}

export interface ListAndDetailsChannelItemProps<
  K extends record.Key,
> extends List.ItemProps<K> {
  port: ReactNode;
  portMaxChars: number;
  icon?: ListAndDetailsIconProps;
  canTare: boolean;
  /** The Synnax channel the item names. Absent when the channel type has none. */
  channel?: channel.Key;
  /** The name deploying the task gives the channel, shown while it has none. */
  defaultName?: string;
  /** Ends the name in the form, as deploying the task ends it. */
  suffix?: string;
  stateChannel?: channel.Key;
  /** The name deploying the task gives the state channel. */
  stateDefaultName?: string;
  /** Deploying the task does not create the channels, because they are disabled. */
  skipped?: boolean;
  onTare?: (channel: channel.Key) => void;
  path: deep.Target<ItemChannel>;
  hasTareButton: boolean;
  nameDirection?: direction.Direction;
}

const getChannelNameProps = (
  hasIcon: boolean,
): Omit<ChannelNameProps, "channel" | "namePath"> => ({
  level: "p",
  color: 9,
  weight: 450,
  className: CSS.cls(
    CSS.BE("channel-item", "name"),
    hasIcon && CSS.BEM("channel-item", "name", "with-icon"),
  ),
  overflow: "fade",
});

export const ListAndDetailsChannelItem = <K extends string>({
  port,
  portMaxChars,
  canTare,
  onTare,
  path,
  hasTareButton,
  channel,
  defaultName,
  suffix,
  icon,
  stateChannel,
  stateDefaultName,
  skipped,
  nameDirection = "x",
  ...rest
}: ListAndDetailsChannelItemProps<K>) => {
  const { itemKey } = rest;
  const hasStateChannel = stateChannel != null;
  const hasIcon = icon != null;
  const item = deep.widen(path);
  const disabled = Form.useFieldValue(item.disabled);
  const channelNameProps = getChannelNameProps(hasIcon);
  const portStyle: CSSProperties = { width: `${portMaxChars * 1.25}rem` };
  return (
    <Select.Item
      {...rest}
      justify="between"
      align="center"
      className={CSS.cls(
        CSS.B("channel-item"),
        disabled && CSS.BM("channel-item", "disabled"),
      )}
    >
      <Flex.Box
        direction={nameDirection}
        gap="small"
        align={nameDirection === "x" ? "center" : "start"}
        className={CSS.BE("channel-item", "label")}
      >
        {/* A lone port element is not an icon, so the port is never square. */}
        <Text.Text
          color={9}
          weight={500}
          style={portStyle}
          className={CSS.BE("channel-item", "port")}
          overflow="fade"
          square={false}
        >
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
        {channel == null ? null : hasStateChannel ? (
          <Flex.Box direction="y" gap="small">
            <WriteChannelNames
              stateNamePath={item.stateChannelName}
              cmdNamePath={item.cmdChannelName}
              cmdChannel={channel}
              cmdDefaultName={defaultName}
              stateChannel={stateChannel}
              stateDefaultName={stateDefaultName}
              skipped={skipped}
              itemKey={itemKey}
            />
          </Flex.Box>
        ) : (
          <ChannelName
            {...channelNameProps}
            channel={channel}
            defaultName={defaultName}
            suffix={suffix}
            skipped={skipped}
            namePath={item.name}
            id={getChannelNameID(itemKey)}
          />
        )}
      </Flex.Box>
      <Flex.Box x align="center" gap="tiny">
        {hasTareButton && (
          <TareButton
            disabled={!canTare}
            onTare={() => channel != null && onTare?.(channel)}
          />
        )}
        <EnabledCheckbox path={item.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};
