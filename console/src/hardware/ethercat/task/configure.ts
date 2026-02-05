// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";

import { Common } from "@/hardware/common";
import { type Device } from "@/hardware/ethercat/device";
import {
  channelMapKey,
  getChannelByMapKey,
  type InputChannel,
  type OutputChannel,
} from "@/hardware/ethercat/task/types";

type Channel = InputChannel | OutputChannel;

export interface SlaveValidationResult<C extends Channel> {
  slaves: Device.SlaveDevice[];
  rack: number;
  channelsBySlaveKey: Map<string, C[]>;
}

export const retrieveAndValidateSlaves = async <C extends Channel>(
  client: Synnax,
  channels: C[],
): Promise<SlaveValidationResult<C>> => {
  const slaveKeys = [...new Set(channels.map((ch) => ch.device))];
  if (slaveKeys.length === 0) throw new Error("No channels configured");

  const slaves = await client.devices.retrieve<
    Device.SlaveProperties,
    Device.Make,
    Device.SlaveModel
  >({ keys: slaveKeys });

  for (const slave of slaves) Common.Device.checkConfigured(slave);

  const networks = [...new Set(slaves.map((s) => s.properties.network))];
  if (networks.length > 1)
    throw new Error(
      `All slaves must be on the same network. Found: ${networks.join(", ")}`,
    );
  if (networks.length === 0 || !networks[0])
    throw new Error("No valid network found for selected slaves");

  const channelsBySlaveKey = new Map<string, C[]>();
  for (const ch of channels) {
    const existing = channelsBySlaveKey.get(ch.device) ?? [];
    existing.push(ch);
    channelsBySlaveKey.set(ch.device, existing);
  }

  return { slaves, rack: slaves[0].rack, channelsBySlaveKey };
};

export interface IndexOptions {
  indexProperty: "readIndex" | "writeStateIndex";
  channelsProperty: "read" | "write";
  nameSuffix: "_time" | "_state_time";
}

export const checkOrCreateIndex = async (
  client: Synnax,
  slave: Device.SlaveDevice,
  options: IndexOptions,
): Promise<boolean> => {
  const { indexProperty, channelsProperty, nameSuffix } = options;
  const currentIndex = slave.properties[indexProperty];
  let shouldCreate = primitive.isZero(currentIndex);
  if (!shouldCreate)
    try {
      await client.channels.retrieve(currentIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreate = true;
      else throw e;
    }

  if (shouldCreate) {
    const identifier = channel.escapeInvalidName(slave.properties.identifier);
    const idx = await client.channels.create({
      name: `${identifier}${nameSuffix}`,
      dataType: "timestamp",
      isIndex: true,
    });
    if (indexProperty === "readIndex") slave.properties.readIndex = idx.key;
    else slave.properties.writeStateIndex = idx.key;
    slave.properties[channelsProperty].channels = {};
    return true;
  }
  return false;
};

export const findChannelsToCreate = async <C extends Channel>(
  client: Synnax,
  channels: C[],
  existingChannels: Record<string, number>,
): Promise<C[]> => {
  const toCreate: C[] = [];
  for (const ch of channels) {
    const mapKey = channelMapKey(ch);
    const existing = getChannelByMapKey(existingChannels, mapKey);
    if (existing === 0) {
      toCreate.push(ch);
      continue;
    }
    try {
      await client.channels.retrieve(existing);
    } catch (e) {
      if (NotFoundError.matches(e)) toCreate.push(ch);
      else throw e;
    }
  }
  return toCreate;
};
