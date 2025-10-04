// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form, Header as PHeader, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { EmptyAction } from "@/components";
import { Common } from "@/hardware/common";
import {
  ChannelList as Core,
  type ChannelListProps as CoreProps,
} from "@/hardware/common/task/ChannelList";
import { type Channel } from "@/hardware/common/task/types";

interface HeaderProps {
  onAdd: () => void;
}

const Header = ({ onAdd }: HeaderProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  return (
    <PHeader.Header>
      <PHeader.Title weight={500} color={10}>
        Channels
      </PHeader.Title>
      {!isSnapshot && (
        <PHeader.Actions>
          <Button.Button
            onClick={onAdd}
            variant="text"
            contrast={2}
            tooltip="Add Channel"
            sharp
          >
            <Icon.Add />
          </Button.Button>
        </PHeader.Actions>
      )}
    </PHeader.Header>
  );
};

interface EmptyContentProps extends HeaderProps {}

const EmptyContent = ({ onAdd }: EmptyContentProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  return (
    <EmptyAction
      message="No channels in task."
      action="Add a channel"
      onClick={isSnapshot ? undefined : onAdd}
    />
  );
};

export interface ChannelListProps<C extends Channel>
  extends Omit<
    CoreProps<C>,
    "data" | "header" | "emptyContent" | "path" | "remove" | "onDuplicate"
  > {
  createChannel: (channels: C[]) => C | null;
  createChannels?: (channels: C[], keys: string[]) => C[];
  path?: string;
}

export const ChannelList = <C extends Channel>({
  createChannel,
  createChannels,
  path = "config.channels",
  ...rest
}: ChannelListProps<C>) => {
  const ctx = Form.useContext();
  const { data, push, remove } = Form.useFieldList<C["key"], C>(path);
  const { onSelect } = rest;
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
      {...rest}
      data={data}
      header={<Header onAdd={handleAdd} />}
      emptyContent={<EmptyContent onAdd={handleAdd} />}
      path={path}
      remove={remove}
      onDuplicate={handleDuplicate}
    />
  );
};
