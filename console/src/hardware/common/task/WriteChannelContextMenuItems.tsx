// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form, Icon, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { deleteWriteChannels } from "@/channel/services/channelDeleteService";
import { renameWriteChannels } from "@/channel/services/channelRenameService";
import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { type WriteChannel } from "@/hardware/common/task/types";
import { useDeleteChannels, useRenameChannels } from "@/modals";

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
  const deleteChannels = useDeleteChannels();
  const { set } = Form.useContext();
  const currentChannels = Form.useFieldValue<WriteChannel[]>("config.channels");
  const queryClient = useQueryClient();

  const handleRenameChannels = useCallback(async () => {
    if (!client) return;
    
    try {
      await renameWriteChannels({ client, channel, renameChannels });
    } catch (error) {
      if (error instanceof Error) 
        handleError(error, "Failed to rename channel(s)");
      
    }
  }, [client, channel, renameChannels, handleError]);

  const handleDeleteChannels = useCallback(async () => {
    if (!client) return;
    
    try {
      // Delete channels from backend
      const wasDeleted = await deleteWriteChannels({ client, channel, deleteChannels });
      
      // Only update UI if deletion was actually confirmed and completed
      if (wasDeleted) {
        // Update form to show "No Channel" immediately after successful deletion
        const updatedChannels = currentChannels.map(ch => {
          if (ch.key === channel.key)
            return {
              ...ch,
              cmdChannel: 0,
              stateChannel: 0,
            };
          return ch;
        });
        set("config.channels", updatedChannels);
        
        // Invalidate all channel queries to refresh Resources pane
        await queryClient.invalidateQueries({ queryKey: ["channel"] });
        await queryClient.invalidateQueries({ queryKey: ["ontology"] });
      }
      
    } catch (error) {
      // If deletion fails due to error, don't update the form
      if (error instanceof Error) 
        handleError(error, "Failed to delete channel(s)");
      
    }
  }, [client, channel, deleteChannels, handleError, currentChannels, set, queryClient]);

  return (
    <>
      <Menu.Item itemKey="renameChannels" onClick={() => void handleRenameChannels()}>
        <Icon.Rename />
        Rename Channels
      </Menu.Item>
      <Menu.Item itemKey="deleteChannels" onClick={() => void handleDeleteChannels()}>
        <Icon.Delete />
        Delete Channels
      </Menu.Item>
    </>
  );
};

export const writeChannelContextMenuItems = Component.renderProp(
  WriteChannelContextMenuItems,
);
