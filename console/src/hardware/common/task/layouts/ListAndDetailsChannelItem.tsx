// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Flex, type List, Select, Text, Tooltip } from "@synnaxlabs/pluto";
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
> extends List.ItemProps<K> {
  port: string | number;
  portMaxChars: number;
  icon?: ListAndDetailsIconProps;
  canTare: boolean;
  channel: channel.Key;
  stateChannel?: channel.Key;
  onTare?: (channel: channel.Key) => void;
  path: string;
  hasTareButton: boolean;
}

const getChannelNameProps = (
  hasIcon: boolean,
): Omit<ChannelNameProps, "channel" | "namePath"> => ({
  level: "p",
  color: 9,
  weight: 450,
  style: {
    maxWidth: hasIcon ? 100 : 150,
    flexGrow: 1,
  },
  overflow: "ellipsis",
});

export const ListAndDetailsChannelItem = <K extends string>({
  port,
  portMaxChars,
  canTare,
  onTare,
  path,
  hasTareButton,
  channel,
  icon,
  stateChannel,
  ...rest
}: ListAndDetailsChannelItemProps<K>) => {
  const { itemKey } = rest;
  const hasStateChannel = stateChannel != null;
  const hasIcon = icon != null;
  const channelNameProps = getChannelNameProps(hasIcon);
  return (
    <Select.ListItem
      {...rest}
      justify="between"
      align="center"
      style={{ padding: "1.25rem 2rem" }}
    >
      <Flex.Box direction="x" gap="small" align="center">
        <Text.Text
          color={8}
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
          <Flex.Box direction="y" gap="small">
            <WriteChannelNames
              stateNamePath={`${path}.stateChannelName`}
              cmdNamePath={`${path}.cmdChannelName`}
              cmdChannel={channel}
              stateChannel={stateChannel}
              itemKey={itemKey}
            />
          </Flex.Box>
        ) : (
          <ChannelName
            {...channelNameProps}
            channel={channel}
            namePath={`${path}.name`}
            id={getChannelNameID(itemKey)}
          />
        )}
      </Flex.Box>
      <Flex.Box pack direction="x" align="center" size="small">
        {hasTareButton && (
          <TareButton disabled={!canTare} onTare={() => onTare?.(channel)} />
        )}
        <EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};
