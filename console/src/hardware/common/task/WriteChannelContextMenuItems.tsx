// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { componentRenderProp, Menu, Text } from "@synnaxlabs/pluto";

import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";
import { type WriteChannel, type WriteChannelType } from "@/hardware/common/task/types";

interface WriteChannelContextMenuItemsProps
  extends ContextMenuItemProps<WriteChannel> {}

export const WriteChannelContextMenuItems: React.FC<
  WriteChannelContextMenuItemsProps
> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const channel = channels.find((ch) => ch.key === key);
  if (!channel) return null;
  const canRenameCmdChannel = channel.cmdChannel !== 0;
  const canRenameStateChannel = channel.stateChannel !== 0;
  const handleRename = (type: WriteChannelType) =>
    Text.edit(getChannelNameID(key, type));
  if (!canRenameCmdChannel && !canRenameStateChannel) return null;
  return (
    <>
      {canRenameCmdChannel && (
        <Menu.Item
          itemKey="renameCmd"
          startIcon={<Icon.Rename />}
          onClick={() => handleRename("cmd")}
        >
          Rename Command Channel
        </Menu.Item>
      )}
      {canRenameStateChannel && (
        <Menu.Item
          itemKey="renameState"
          startIcon={<Icon.Rename />}
          onClick={() => handleRename("state")}
        >
          Rename State Channel
        </Menu.Item>
      )}
      <Menu.Divider />
    </>
  );
};

export const writeChannelContextMenuItems = componentRenderProp(
  WriteChannelContextMenuItems,
);
