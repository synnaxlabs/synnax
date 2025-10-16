// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, ContextMenu as PContextMenu, Text } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/components";
import { type ContextMenuItemProps } from "@/hardware/common/task/ChannelList";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";
import { type ReadChannel } from "@/hardware/common/task/types";

export interface ReadChannelContextMenuItemProps
  extends ContextMenuItemProps<ReadChannel> {}

export const ReadChannelContextMenuItem: React.FC<ReadChannelContextMenuItemProps> = ({
  channels,
  keys,
}) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const channel = channels.find((ch) => ch.key === key)?.channel;
  if (channel == null || channel == 0) return null;
  const handleRename = () => Text.edit(getChannelNameID(key));
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <PContextMenu.Divider />
    </>
  );
};

export const readChannelContextMenuItem = Component.renderProp(
  ReadChannelContextMenuItem,
);
