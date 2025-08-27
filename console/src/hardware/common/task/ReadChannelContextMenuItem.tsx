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

import { deleteReadChannel } from "@/channel/services/channelDeleteService";
import { renameReadChannel } from "@/channel/services/channelRenameService";
import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { type ReadChannel } from "@/hardware/common/task/types";
import { useDeleteChannels, useRenameChannels } from "@/modals";

export interface ReadChannelContextMenuItemProps
  extends ContextMenuItemProps<ReadChannel> {}

export const ReadChannelContextMenuItem: React.FC<ReadChannelContextMenuItemProps> = ({
  channels,
  keys,
}) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const channel = channels.find((ch) => ch.key === key)?.channel;
  const canRename = channel != null && channel !== 0;
  
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const renameChannels = useRenameChannels();
  const deleteChannels = useDeleteChannels();
  const { set } = Form.useContext();
  const currentChannels = Form.useFieldValue<ReadChannel[]>("config.channels");
  const queryClient = useQueryClient();

  const handleRename = useCallback(async () => {
    if (!canRename || !client) return;
    
    try {
      await renameReadChannel({ client, channelKey: channel, renameChannels });
    } catch (error) {
      if (error instanceof Error) 
        handleError(error, "Failed to rename channel");
      
    }
  }, [canRename, channel, client, renameChannels, handleError]);

  const handleDelete = useCallback(async () => {
    if (!canRename || !client) return;
    
    try {
      const wasDeleted = await deleteReadChannel({ client, channelKey: channel, deleteChannels });
      
      // Only update UI if deletion was actually confirmed and completed
      if (wasDeleted) {
        // Update form to show "No Channel" immediately after successful deletion
        const updatedChannels = currentChannels.map(ch => {
          if (ch.key === key)
            return {
              ...ch,
              channel: 0,
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
        handleError(error, "Failed to delete channel");
      
    }
  }, [canRename, channel, client, deleteChannels, handleError, currentChannels, set, key, queryClient]);

  return (
    <>
      {canRename && (
        <Menu.Item itemKey="rename" onClick={() => void handleRename()}>
          <Icon.Rename />
          Rename
        </Menu.Item>
      )}
      {canRename && (
        <Menu.Item itemKey="delete" onClick={() => void handleDelete()}>
          <Icon.Delete />
          Delete
        </Menu.Item>
      )}
    </>
  );
};

export const readChannelContextMenuItem = Component.renderProp(
  ReadChannelContextMenuItem,
);
