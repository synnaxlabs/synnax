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
import { type ReadChannel } from "@/hardware/common/task/types";

export interface ReadChannelContextMenuItemProps extends ContextMenuItemProps<ReadChannel> {}

export const ReadChannelContextMenuItem: React.FC<ReadChannelContextMenuItemProps> = ({
  keys,
}) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const handleRename = () => Text.edit(getChannelNameID(key));
  return (
    <>
      <Menu.Item itemKey="rename" onClick={handleRename}>
        <Icon.Rename />
        Rename
      </Menu.Item>
      <Menu.Divider />
    </>
  );
};

export const readChannelContextMenuItem = Component.renderProp(
  ReadChannelContextMenuItem,
);
