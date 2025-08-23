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

import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { renameReadChannel } from "@/hardware/common/task/channelRenameService";
import { type ReadChannel } from "@/hardware/common/task/types";
import { useRenameChannels } from "@/modals";

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

  const handleRename = useCallback(async () => {
    if (!canRename || !client) return;
    
    try {
      await renameReadChannel({ client, channelKey: channel, renameChannels });
    } catch (error) {
      if (error instanceof Error) 
        handleError(error, "Failed to rename channel");
      
    }
  }, [canRename, channel, client, renameChannels, handleError]);

  return (
    <>
      {canRename && (
        <Menu.Item itemKey="rename" onClick={() => void handleRename()}>
          <Icon.Rename />
          Rename
        </Menu.Item>
      )}
    </>
  );
};

export const readChannelContextMenuItem = Component.renderProp(
  ReadChannelContextMenuItem,
);
