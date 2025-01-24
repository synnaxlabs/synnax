// Copyright 2024 Synnax Labs, Inc.
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
  type Channel,
  ChannelList,
  type ChannelListProps,
} from "@/hardware/common/task/ChannelList";

interface HeaderProps {
  isSnapshot: boolean;
  onAdd: () => void;
}

const Header = ({ isSnapshot, onAdd }: HeaderProps) => (
  <PHeader.Header level="h4">
    <PHeader.Title weight={500}>Channels</PHeader.Title>
    {!isSnapshot && (
      <PHeader.Actions>
        {[{ key: "add", onClick: onAdd, children: <Icon.Add />, size: "large" }]}
      </PHeader.Actions>
    )}
  </PHeader.Header>
);

interface EmptyContentProps extends HeaderProps {}

const EmptyContent = ({ isSnapshot, onAdd }: EmptyContentProps) => (
  <Align.Center direction="y" justify="center">
    <Text.Text level="p">No channels in task.</Text.Text>
    {!isSnapshot && (
      <Text.Link level="p" onClick={onAdd}>
        Add a channel
      </Text.Link>
    )}
  </Align.Center>
);

export type DefaultChannelListProps<C extends Channel> = Omit<
  ChannelListProps<C>,
  "channels" | "header" | "emptyContent" | "path" | "remove"
> & {
  generateChannel: (channels: C[]) => C;
  path?: string;
};

export const DefaultChannelList = <C extends Channel>({
  isSnapshot,
  children,
  generateChannel,
  onSelect,
  path = "config.channels",
  ...rest
}: DefaultChannelListProps<C>) => {
  const {
    value: channels,
    push,
    remove,
  } = Form.useFieldArray<C>({ path, updateOnChildren: true });
  const handleAdd = useCallback(() => {
    const channel = generateChannel(channels);
    push(channel);
    onSelect([channel.key], channels.length);
  }, [push, channels, generateChannel, onSelect]);
  return (
    <ChannelList
      header={<Header isSnapshot={isSnapshot} onAdd={handleAdd} />}
      emptyContent={<EmptyContent isSnapshot={isSnapshot} onAdd={handleAdd} />}
      isSnapshot={isSnapshot}
      channels={channels}
      path={path}
      remove={remove}
      onSelect={onSelect}
      {...rest}
    >
      {(p) => children(p)}
    </ChannelList>
  );
};
