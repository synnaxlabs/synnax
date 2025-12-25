// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Icon, Menu, Text } from "@synnaxlabs/pluto";

import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";
import { type WriteChannel, type WriteChannelType } from "@/hardware/common/task/types";

export interface WriteChannelContextMenuItemsProps extends ContextMenuItemProps<WriteChannel> {}

export const WriteChannelContextMenuItems: React.FC<
  WriteChannelContextMenuItemsProps
> = ({ channels, keys }) => {
  const key = keys[0];
  const channel = channels.find((ch) => ch.key === key);
  if (channel == null) return null;
  const handleRename = (type: WriteChannelType) =>
    Text.edit(getChannelNameID(key, type));
  return (
    <>
      <Menu.Item itemKey="renameCmd" onClick={() => handleRename("cmd")}>
        <Icon.Rename />
        Rename command channel
      </Menu.Item>
      <Menu.Item itemKey="renameState" onClick={() => handleRename("state")}>
        <Icon.Rename />
        Rename state channel
      </Menu.Item>
      <Menu.Divider />
    </>
  );
};

export const writeChannelContextMenuItems = Component.renderProp(
  WriteChannelContextMenuItems,
);
