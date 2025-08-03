// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Header as PHeader, Icon, Text } from "@synnaxlabs/pluto";
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
  <PHeader.Header>
    <PHeader.Title weight={500} color={10}>
      Channels
    </PHeader.Title>
    {!isSnapshot && (
      <PHeader.Actions>
        {[
          {
            key: "add",
            onClick: onAdd,
            children: <Icon.Add />,
            variant: "text",
            size: "medium",
            contrast: 2,
            sharp: true,
            tooltip: "Add Channel",
          },
        ]}
      </PHeader.Actions>
    )}
  </PHeader.Header>
);

interface EmptyContentProps extends HeaderProps {}

const EmptyContent = ({ isSnapshot, onAdd }: EmptyContentProps) => (
  <Flex.Box grow center>
    <Text.Text>No channels in task.</Text.Text>
    {!isSnapshot && <Text.Text onClick={onAdd}>Add a channel</Text.Text>}
  </Flex.Box>
);

export interface ChannelListProps<C extends Channel>
  extends Omit<
    CoreProps<C>,
    "data" | "header" | "emptyContent" | "path" | "remove" | "useListItem" | "value"
  > {
  createChannel: (channels: C[]) => C | null;
  createChannels?: (channels: C[], keys: string[]) => C[];
  path?: string;
}

export const ChannelList = <C extends Channel>({
  isSnapshot,
  createChannel,
  createChannels,
  onSelect,
  path = "config.channels",
  listItem,
  selected,
}: ChannelListProps<C>) => {
  const ctx = Form.useContext();
  const { data, push, remove } = Form.useFieldList<C["key"], C>(path);
  const handleAdd = useCallback(() => {
    const channels = ctx.get<C[]>(path).value;
    const channel = createChannel(channels);
    if (channel == null) return;
    push(channel);
    onSelect([channel.key]);
  }, [push, createChannel, onSelect]);
  const handleDuplicate = useCallback(
    (chs: C[], keys: string[]) => {
      if (createChannels == null) return;
      const duplicated = createChannels(chs, keys);
      push(duplicated);
      return duplicated.map(({ key }) => key);
    },
    [createChannels, onSelect, push],
  );
  return (
    <Core
      header={<Header isSnapshot={isSnapshot} onAdd={handleAdd} />}
      emptyContent={<EmptyContent isSnapshot={isSnapshot} onAdd={handleAdd} />}
      isSnapshot={isSnapshot}
      path={path}
      onSelect={onSelect}
      onDuplicate={handleDuplicate}
      data={data}
      listItem={listItem}
      selected={selected}
      remove={remove}
    />
  );
};
