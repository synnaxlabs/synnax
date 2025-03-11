// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Form, Header as PHeader, Text } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import {
  ChannelList as Core,
  type ChannelListProps as CoreProps,
} from "@/hardware/common/task/ChannelList";
import { type Channel } from "@/hardware/common/task/types";

interface HeaderProps {
  isSnapshot: boolean;
  onAdd: () => void;
}

const Header = ({ isSnapshot, onAdd }: HeaderProps) => (
  <PHeader.Header level="p">
    <PHeader.Title weight={500} shade={8}>
      Channels
    </PHeader.Title>
    {!isSnapshot && (
      <PHeader.Actions>
        {[
          {
            key: "add",
            onClick: onAdd,
            children: <Icon.Add />,
            size: "small",
            tooltip: "Add Channel",
          },
        ]}
      </PHeader.Actions>
    )}
  </PHeader.Header>
);

interface EmptyContentProps extends HeaderProps {}

const EmptyContent = ({ isSnapshot, onAdd }: EmptyContentProps) => (
  <Align.Center grow>
    <Text.Text level="p">No channels in task.</Text.Text>
    {!isSnapshot && (
      <Text.Link level="p" onClick={onAdd}>
        Add a channel
      </Text.Link>
    )}
  </Align.Center>
);

export type ChannelListProps<C extends Channel> = Omit<
  CoreProps<C>,
  "channels" | "header" | "emptyContent" | "path" | "remove"
> & {
  createChannel: (channels: C[]) => C | null;
  path?: string;
};

export const ChannelList = <C extends Channel>({
  isSnapshot,
  createChannel,
  onSelect,
  path = "config.channels",
  ...rest
}: ChannelListProps<C>) => {
  const {
    value: channels,
    push,
    remove,
  } = Form.useFieldArray<C>({ path, updateOnChildren: true });
  const handleAdd = useCallback(() => {
    const channel = createChannel(channels);
    if (channel == null) return;
    push(channel);
    onSelect([channel.key], channels.length);
  }, [push, channels, createChannel, onSelect]);
  return (
    <Core
      header={<Header isSnapshot={isSnapshot} onAdd={handleAdd} />}
      emptyContent={<EmptyContent isSnapshot={isSnapshot} onAdd={handleAdd} />}
      isSnapshot={isSnapshot}
      channels={channels}
      path={path}
      remove={remove}
      onSelect={onSelect}
      {...rest}
    />
  );
};
