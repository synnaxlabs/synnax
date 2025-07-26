// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Align, List, Text, Tooltip } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { cloneElement, type JSX } from "react";

import { ChannelName, type ChannelNameProps } from "@/hardware/common/task/ChannelName";
import { EnableDisableButton } from "@/hardware/common/task/EnableDisableButton";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";
import { TareButton } from "@/hardware/common/task/TareButton";
import { WriteChannelNames } from "@/hardware/common/task/WriteChannelNames";

export interface ListAndDetailsIconProps {
  icon: JSX.Element;
  name: string;
}

export interface ListAndDetailsChannelItemProps<
  K extends record.Key,
  E extends record.Keyed<K>,
> extends List.ItemProps<K, E> {
  port: string | number;
  portMaxChars: number;
  icon?: ListAndDetailsIconProps;
  canTare: boolean;
  channel: channel.Key;
  stateChannel?: channel.Key;
  onTare?: (channel: channel.Key) => void;
  isSnapshot: boolean;
  path: string;
  hasTareButton: boolean;
}

const getChannelNameProps = (hasIcon: boolean): Omit<ChannelNameProps, "channel"> => ({
  level: "p",
  shade: 9,
  weight: 450,
  style: {
    maxWidth: hasIcon ? 100 : 150,
    flexGrow: 1,
    textOverflow: "ellipsis",
    overflow: "hidden",
  },
  noWrap: true,
});

export const ListAndDetailsChannelItem = <K extends string, E extends record.Keyed<K>>({
  port,
  portMaxChars,
  canTare,
  onTare,
  isSnapshot,
  path,
  hasTareButton,
  channel,
  icon,
  stateChannel,
  ...rest
}: ListAndDetailsChannelItemProps<K, E>) => {
  const { key } = rest.entry;
  const hasStateChannel = stateChannel != null;
  const hasIcon = icon != null;
  const channelNameProps = getChannelNameProps(hasIcon);
  return (
    <List.ItemFrame
      {...rest}
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.25rem 2rem" }}
    >
      <Align.Space direction="x" gap="small" align="center">
        <Text.Text
          level="p"
          shade={8}
          weight={500}
          style={{ width: `${portMaxChars * 1.25}rem` }}
        >
          {port}
        </Text.Text>
        {hasIcon && (
          <Tooltip.Dialog>
            {icon.name}
            {cloneElement(icon.icon, {
              style: {
                height: "var(--pluto-p-size)",
                fontSize: "var(--pluto-p-size)",
                color: "var(--pluto-gray-l8)",
              },
            })}
          </Tooltip.Dialog>
        )}
        {hasStateChannel ? (
          <Align.Space direction="y" gap="small">
            <WriteChannelNames
              cmdChannel={channel}
              stateChannel={stateChannel}
              itemKey={key}
            />
          </Align.Space>
        ) : (
          <ChannelName
            {...channelNameProps}
            channel={channel}
            id={getChannelNameID(key)}
          />
        )}
      </Align.Space>
      <Align.Pack direction="x" align="center" gap="small">
        {hasTareButton && (
          <TareButton disabled={!canTare} onTare={() => onTare?.(channel)} />
        )}
        <EnableDisableButton path={`${path}.enabled`} isSnapshot={isSnapshot} />
      </Align.Pack>
    </List.ItemFrame>
  );
};
