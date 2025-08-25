// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

import { cleanChannelName, extractBaseName } from "@/channel/services/channelNameUtils";
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

export interface RenameWriteChannelsByPatternParams {
  client: Synnax;
  baseName: string;
  renameChannels: ReturnType<typeof useRenameChannels>;
}

export const renameWriteChannelsByPattern = async ({
  client,
  baseName,
  renameChannels,
}: RenameWriteChannelsByPatternParams): Promise<void> => {
  // Try to retrieve each related channel by name
  const channelNames = [`${baseName}_cmd`, `${baseName}_state`, `${baseName}_cmd_time`];
  const relatedChannels: any[] = [];
  
  for (const name of channelNames) 
    try {
      const channel = await client.channels.retrieve(name);
      relatedChannels.push(channel);
    } catch (_e) {
      // Channel doesn't exist, skip it
    }


  const cmdChannel = relatedChannels.find(ch => ch.name === `${baseName}_cmd`);
  const stateChannel = relatedChannels.find(ch => ch.name === `${baseName}_state`);
  const cmdTimeChannel = relatedChannels.find(ch => ch.name === `${baseName}_cmd_time`);

  const canRenameCmdChannel = cmdChannel != null;
  const canRenameStateChannel = stateChannel != null;

  // Build the old names array
  const oldNames: string[] = [];
  if (cmdChannel) oldNames.push(cmdChannel.name);
  if (stateChannel) oldNames.push(stateChannel.name);
  if (cmdTimeChannel) oldNames.push(cmdTimeChannel.name);

  const newBaseName = await renameChannels(
    {
      initialValue: baseName,
      allowEmpty: false,
      label: "Base Channel Name",
      oldNames,
      canRenameCmdChannel,
      canRenameStateChannel,
    },
    {
      icon: "Rename",
      name: "Rename Channels",
    }
  );

  if (!newBaseName) return;

  // Rename all found channels
  const renamePromises = [];
  if (cmdChannel) {
    renamePromises.push(client.channels.rename(cmdChannel.key, `${newBaseName}_cmd`));
    if (cmdTimeChannel) 
      renamePromises.push(client.channels.rename(cmdTimeChannel.key, `${newBaseName}_cmd_time`));
  }
  if (stateChannel) 
    renamePromises.push(client.channels.rename(stateChannel.key, `${newBaseName}_state`));

  await Promise.all(renamePromises);
};

export const renameWriteChannels = async ({
  client,
  channel,
  renameChannels,
}: RenameWriteChannelParams): Promise<void> => {
  const cmdChannel = channel.cmdChannel;
  const stateChannel = channel.stateChannel;
  
  const canRenameCmdChannel = cmdChannel !== 0;
  const canRenameStateChannel = stateChannel !== 0;
  
  const channelPromises = [
    ...(canRenameCmdChannel ? [client.channels.retrieve(cmdChannel)] : []),
    ...(canRenameStateChannel ? [client.channels.retrieve(stateChannel)] : [])
  ];
  
  const channelDetails = await Promise.all(channelPromises);
  
  // Build the names arrays carefully based on the order of retrieval
  const oldNames: string[] = [];
  
  let cmdChannelDetail = null;
  let stateChannelDetail = null;
  
  if (canRenameCmdChannel && canRenameStateChannel) {
    // Both channels exist - cmd is first, state is second
    cmdChannelDetail = channelDetails[0];
    stateChannelDetail = channelDetails[1];
    oldNames.push(cmdChannelDetail.name);
    oldNames.push(stateChannelDetail.name);
  } else if (canRenameCmdChannel) {
    // Only cmd channel exists
    cmdChannelDetail = channelDetails[0];
    oldNames.push(cmdChannelDetail.name);
  } else if (canRenameStateChannel) {
    // Only state channel exists
    stateChannelDetail = channelDetails[0];
    oldNames.push(stateChannelDetail.name);
  }
  
  // Always include cmd_time channel in oldNames when cmd channel exists
  if (canRenameCmdChannel && cmdChannelDetail) 
    try {
      const timeChannel = await client.channels.retrieve(cmdChannelDetail.index);
      oldNames.push(timeChannel.name);
    } catch (_e) {
      // cmd_time channel not found
    }
  
  const initialValue = channel.customName ? extractBaseName(channel.customName) : "";
  
  const newBaseName = await renameChannels(
    {
      initialValue,
      allowEmpty: false,
      label: "Base Channel Name",
      oldNames,
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
  
  if (canRenameCmdChannel && cmdChannelDetail) {
    renamePromises.push(client.channels.rename(cmdChannel, `${newBaseName}_cmd`));
    try {
      renamePromises.push(client.channels.rename(cmdChannelDetail.index, `${newBaseName}_cmd_time`));
    } catch (_e) {
      // cmd_time channel rename failed
    }
  }
  
  if (canRenameStateChannel)
    renamePromises.push(client.channels.rename(stateChannel, `${newBaseName}_state`));
  
  await Promise.all(renamePromises);
};