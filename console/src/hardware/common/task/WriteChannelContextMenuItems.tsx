// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Icon, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { renameWriteChannels } from "@/channel/services/channelRenameService";
import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { type WriteChannel } from "@/hardware/common/task/types";
import { useRenameChannels } from "@/modals";

export interface WriteChannelContextMenuItemsProps
  extends ContextMenuItemProps<WriteChannel> {}

export const WriteChannelContextMenuItems: React.FC<
  WriteChannelContextMenuItemsProps
> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const channel = channels.find((ch) => ch.key === key);
  if (channel == null) return null;
  
  const canRenameCmdChannel = channel.cmdChannel !== 0;
  const canRenameStateChannel = channel.stateChannel !== 0;
  if (!canRenameCmdChannel && !canRenameStateChannel) return null;
  
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const renameChannels = useRenameChannels();

  const handleRenameChannels = useCallback(async () => {
    if (!client) return;
    
    try {
      await renameWriteChannels({ client, channel, renameChannels });
    } catch (error) {
      if (error instanceof Error) 
        handleError(error, "Failed to rename channel(s)");
      
    }
  }, [client, channel, renameChannels, handleError]);

  return (
    <>
      <Menu.Item itemKey="renameChannels" onClick={() => void handleRenameChannels()}>
        <Icon.Rename />
        Rename Channels
      </Menu.Item>
    </>
  );
};

export const writeChannelContextMenuItems = Component.renderProp(
  WriteChannelContextMenuItems,
);
