// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

import { cleanChannelName, extractBaseName } from "@/hardware/common/task/channelNameUtils";
import { type WriteChannel } from "@/hardware/common/task/types";
import { type useRenameChannels } from "@/modals";

export interface RenameReadChannelParams {
  client: Synnax;
  channelKey: number;
  renameChannels: ReturnType<typeof useRenameChannels>;
}

export interface RenameWriteChannelParams {
  client: Synnax;
  channel: WriteChannel;
  renameChannels: ReturnType<typeof useRenameChannels>;
}

export const renameReadChannel = async ({
  client,
  channelKey,
  renameChannels,
}: RenameReadChannelParams): Promise<void> => {
  const channelDetail = await client.channels.retrieve(channelKey);
  
  const newChannelName = await renameChannels(
    {
      initialValue: channelDetail.name,
      allowEmpty: false,
      label: "Channel Name",
      oldNames: [channelDetail.name],
      currentNames: [channelDetail.name],
      canRenameCmdChannel: false,
      canRenameStateChannel: false,
    },
    {
      icon: "Rename",
      name: "Rename Channel",
    }
  );
  
  if (!newChannelName) return;
  
  const cleanedName = cleanChannelName(newChannelName.trim());
  if (cleanedName !== "")
    await client.channels.rename(channelKey, cleanedName);
};

export const renameWriteChannels = async ({
  client,
  channel,
  renameChannels,
}: RenameWriteChannelParams): Promise<void> => {
  const canRenameCmdChannel = channel.cmdChannel !== 0;
  const canRenameStateChannel = channel.stateChannel !== 0;
  
  const channelPromises = [
    ...(canRenameCmdChannel ? [client.channels.retrieve(channel.cmdChannel)] : []),
    ...(canRenameStateChannel ? [client.channels.retrieve(channel.stateChannel)] : [])
  ];
  
  const channelDetails = await Promise.all(channelPromises);
  const currentNames = channelDetails.map(ch => ch.name);
  const oldNames = [...currentNames];
  
  // Always include cmd_time channel in oldNames when cmd channel exists
  if (canRenameCmdChannel) {
    const cmdChannel = channelDetails[0];
    const timeChannel = await client.channels.retrieve(cmdChannel.index);
    oldNames.push(timeChannel.name);
    currentNames.push(timeChannel.name);
  }
  
  const initialValue = channel.customName ? extractBaseName(channel.customName) : "";
  
  const newBaseName = await renameChannels(
    {
      initialValue,
      allowEmpty: false,
      label: "Base Channel Name",
      oldNames,
      currentNames,
      canRenameCmdChannel,
      canRenameStateChannel,
    },
    {
      icon: "Rename",
      name: "Rename Channels",
    }
  );
  
  if (!newBaseName) return;
  
  const renamePromises = [];
  
  if (canRenameCmdChannel) {
    const cmdChannel = channelDetails[0];
    renamePromises.push(client.channels.rename(channel.cmdChannel, `${newBaseName}_cmd`));
    renamePromises.push(client.channels.rename(cmdChannel.index, `${newBaseName}_cmd_time`));
  }
  
  if (canRenameStateChannel)
    renamePromises.push(client.channels.rename(channel.stateChannel, `${newBaseName}_state`));
  
  await Promise.all(renamePromises);
};