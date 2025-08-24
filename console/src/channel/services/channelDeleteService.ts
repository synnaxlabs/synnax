// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";

import { type WriteChannel } from "@/hardware/common/task/types";
import { type useDeleteChannels } from "@/modals";

export interface DeleteMultipleChannelsParams {
  client: Synnax;
  channelKeys: number[];
  deleteChannels: ReturnType<typeof useDeleteChannels>;
}

export interface DeleteReadChannelParams {
  client: Synnax;
  channelKey: number;
  deleteChannels: ReturnType<typeof useDeleteChannels>;
}

export interface DeleteWriteChannelParams {
  client: Synnax;
  channel: WriteChannel;
  deleteChannels: ReturnType<typeof useDeleteChannels>;
}

export const deleteReadChannel = async ({
  client,
  channelKey,
  deleteChannels,
}: DeleteReadChannelParams): Promise<boolean> => {
  const channelDetail = await client.channels.retrieve(channelKey);
  
  const confirmed = await deleteChannels(
    {
      message: `Are you sure you want to delete ${channelDetail.name}?`,
      channelNames: [channelDetail.name],
    },
    {
      icon: "Delete",
      name: "Delete Channel",
    }
  );
  
  if (!confirmed) return false;
  
  await client.channels.delete(channelKey);
  return true;
};

export interface DeleteWriteChannelsByPatternParams {
  client: Synnax;
  baseName: string;
  deleteChannels: ReturnType<typeof useDeleteChannels>;
}

export const deleteWriteChannelsByPattern = async ({
  client,
  baseName,
  deleteChannels,
}: DeleteWriteChannelsByPatternParams): Promise<void> => {
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

  if (relatedChannels.length === 0) return;

  // Build the channel names array for display
  const channelNamesToDelete: string[] = [];
  if (cmdChannel) channelNamesToDelete.push(cmdChannel.name);
  if (stateChannel) channelNamesToDelete.push(stateChannel.name);
  if (cmdTimeChannel) channelNamesToDelete.push(cmdTimeChannel.name);

  const confirmed = await deleteChannels(
    {
      message: `Are you sure you want to delete ${baseName}?`,
      channelNames: channelNamesToDelete,
    },
    {
      icon: "Delete",
      name: "Delete Channels",
    }
  );

  if (!confirmed) return;

  // Delete all found channels
  const channelKeysToDelete = relatedChannels.map(ch => ch.key);
  await client.channels.delete(channelKeysToDelete);
};

export const deleteWriteChannels = async ({
  client,
  channel,
  deleteChannels,
}: DeleteWriteChannelParams): Promise<boolean> => {
  const cmdChannel = channel.cmdChannel;
  const stateChannel = channel.stateChannel;
  
  const canDeleteCmdChannel = cmdChannel !== 0;
  const canDeleteStateChannel = stateChannel !== 0;
  
  const channelPromises = [
    ...(canDeleteCmdChannel ? [client.channels.retrieve(cmdChannel)] : []),
    ...(canDeleteStateChannel ? [client.channels.retrieve(stateChannel)] : [])
  ];
  
  const channelDetails = await Promise.all(channelPromises);
  
  // Build the names arrays carefully based on the order of retrieval
  const channelNamesToDelete: string[] = [];
  const channelKeysToDelete: number[] = [];
  
  let cmdChannelDetail = null;
  let stateChannelDetail = null;
  
  if (canDeleteCmdChannel && canDeleteStateChannel) {
    // Both channels exist - cmd is first, state is second
    cmdChannelDetail = channelDetails[0];
    stateChannelDetail = channelDetails[1];
    channelNamesToDelete.push(cmdChannelDetail.name);
    channelNamesToDelete.push(stateChannelDetail.name);
    channelKeysToDelete.push(cmdChannel);
    channelKeysToDelete.push(stateChannel);
  } else if (canDeleteCmdChannel) {
    // Only cmd channel exists
    cmdChannelDetail = channelDetails[0];
    channelNamesToDelete.push(cmdChannelDetail.name);
    channelKeysToDelete.push(cmdChannel);
  } else if (canDeleteStateChannel) {
    // Only state channel exists
    stateChannelDetail = channelDetails[0];
    channelNamesToDelete.push(stateChannelDetail.name);
    channelKeysToDelete.push(stateChannel);
  }
  
  // Always include cmd_time channel in deletion when cmd channel exists
  if (canDeleteCmdChannel && cmdChannelDetail) 
    try {
      const timeChannel = await client.channels.retrieve(cmdChannelDetail.index);
      channelNamesToDelete.push(timeChannel.name);
      channelKeysToDelete.push(cmdChannelDetail.index);
    } catch (_e) {
      // cmd_time channel not found
    }
  
  if (channelKeysToDelete.length === 0) return false;

  const confirmed = await deleteChannels(
    {
      message: `Are you sure you want to delete ${channel.customName}?`,
      channelNames: channelNamesToDelete,
    },
    {
      icon: "Delete",
      name: "Delete Channels",
    }
  );
  
  if (!confirmed) return false;
  
  await client.channels.delete(channelKeysToDelete);
  return true;
};

export const deleteMultipleChannels = async ({
  client,
  channelKeys,
  deleteChannels,
}: DeleteMultipleChannelsParams): Promise<void> => {
  if (channelKeys.length === 0) return;
  
  const channelDetails = await Promise.all(
    channelKeys.map(key => client.channels.retrieve(key))
  );
  
  const channelNames = channelDetails.map(channel => channel.name);
  const channelWord = channelKeys.length === 1 ? 'channel' : 'channels';
  
  const confirmed = await deleteChannels(
    {
      message: `Are you sure you want to delete ${channelKeys.length} ${channelWord}?`,
      channelNames,
    },
    {
      icon: "Delete",
      name: "Delete Channels",
    }
  );
  
  if (!confirmed) return;
  
  await client.channels.delete(channelKeys);
};