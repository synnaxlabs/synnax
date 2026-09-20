// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Access, Component, Menu, Text } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { type ContextMenuItemProps } from "@/platform/task/ChannelList";
import { getChannelNameID } from "@/platform/task/getChannelNameID";
import { type Channel } from "@/platform/task/types";

export interface ReadChannelContextMenuItemProps extends ContextMenuItemProps<Channel> {}

export const ReadChannelContextMenuItem: React.FC<ReadChannelContextMenuItemProps> = ({
  keys,
}) => {
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  if (keys.length !== 1 || !canRename) return null;
  const key = keys[0];
  const handleRename = () => Text.edit(getChannelNameID(key));
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <Menu.Divider />
    </>
  );
};

export const readChannelContextMenuItem = Component.renderProp(
  ReadChannelContextMenuItem,
);
