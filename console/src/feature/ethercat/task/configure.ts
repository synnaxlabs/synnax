// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, ethercat, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { DataType, errors, primitive } from "@synnaxlabs/x";

import { Device } from "@/feature/ethercat/device";
import {
  addressKey,
  DIRECTIONS,
  entryAt,
  type EntryChannel,
  getUnresolvedMessage,
  hasEntry,
  type PDOType,
  type ReadChannel,
  type WriteChannel,
} from "@/feature/ethercat/task/types";
import { Device as PlatformDevice } from "@/platform/device";

type Channel = ReadChannel | WriteChannel;

export interface SlaveValidationResult<C extends Channel> {
  slaves: Device.SlaveDevice[];
  rack: number;
  channelsBySlaveKey: Map<string, Extract<C, EntryChannel>[]>;
}

/** @returns the size in bytes of the number type that holds an entry. */
const entryWidth = (bitLength: number): number => {
  if (bitLength <= 8) return 1;
  if (bitLength <= 16) return 2;
  if (bitLength <= 32) return 4;
  return 8;
};

const validateEntries = <C extends Channel>(
  slave: Device.SlaveDevice,
  channels: C[],
  pdoType: PDOType,
): Extract<C, EntryChannel>[] => {
  const used = new Set<string>();
  const entries = slave.properties.pdos[pdoType];
  return channels
    .filter((ch) => !ch.disabled)
    .map((ch) => {
      if (ch.type === "unresolved")
        throw new Error(getUnresolvedMessage(ch.pdo, slave, pdoType));
      if (!hasEntry(ch))
        throw new Error(
          `A channel on slave ${slave.name} has no entry. Pick an entry.`,
        );
      const address = ethercat.formatAddress(ch);
      const entry = entryAt(entries, ch);
      if (entry == null)
        throw new Error(
          `Slave ${slave.name} has no ${DIRECTIONS[pdoType]} entry at ${address}. ` +
            "Pick an entry.",
        );
      const dataType = new DataType(ch.dataType);
      const width = entryWidth(entry.bitLength);
      if (
        !(dataType.isInteger || dataType.isFloat) ||
        dataType.density.valueOf() !== width
      )
        throw new Error(
          `Data type ${ch.dataType} does not fit entry ${address} on slave ` +
            `${slave.name}, which holds ${entry.bitLength} bits. Pick a ${width} ` +
            "byte number type.",
        );
      const key = addressKey(ch);
      if (used.has(key))
        throw new Error(
          `More than one channel uses entry ${address} on slave ${slave.name}. ` +
            "Pick a different entry for one of them.",
        );
      used.add(key);
      return ch;
    });
};

/**
 * Retrieves the slaves the channels use and checks the entries of the enabled channels.
 * @returns the slaves, their rack, and the enabled channels of each slave.
 * @throws {Error} if the slaves are on different networks, or if an enabled channel has
 * no entry, an entry its slave lacks, or an entry another channel uses.
 */
export const retrieveAndValidateSlaves = async <C extends Channel>(
  client: Synnax,
  channels: C[],
  pdoType: PDOType,
): Promise<SlaveValidationResult<C>> => {
  const slaveKeys = [...new Set(channels.map((ch) => ch.device))];
  const slaves = await client.devices.retrieve({
    keys: slaveKeys,
    schemas: Device.SLAVE_SCHEMAS,
  });

  for (const slave of slaves) PlatformDevice.checkConfigured(slave);

  const networks = [...new Set(slaves.map((s) => s.properties.network))];
  if (networks.length > 1)
    throw new Error(
      `All slaves must be on the same network. Found: ${networks.join(", ")}. ` +
        "Pick slaves on one network.",
    );
  if (networks.length === 0 || !networks[0])
    throw new Error(
      "No network found for the selected slaves. Scan the network again.",
    );

  const channelsBySlaveKey = new Map<string, Extract<C, EntryChannel>[]>();
  for (const slave of slaves)
    channelsBySlaveKey.set(
      slave.key,
      validateEntries(
        slave,
        channels.filter((ch) => ch.device === slave.key),
        pdoType,
      ),
    );

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
      else throw errors.fromUnknown(e);
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

/**
 * @returns the channels whose mapped Synnax channel is missing.
 * @throws {Error} if a mapped channel holds a different data type than its channel.
 */
export const findChannelsToCreate = async <C extends EntryChannel>(
  client: Synnax,
  channels: C[],
  existingChannels: Record<string, number>,
  mapKey: (ch: C) => string = addressKey,
): Promise<C[]> => {
  const toCreate: C[] = [];
  for (const ch of channels) {
    const existing = existingChannels[mapKey(ch)] ?? 0;
    if (existing === 0) {
      toCreate.push(ch);
      continue;
    }
    let mapped: channel.Channel;
    try {
      mapped = await client.channels.retrieve(existing);
    } catch (e) {
      if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      toCreate.push(ch);
      continue;
    }
    const dataType = mapped.dataType.toString();
    if (dataType !== ch.dataType)
      throw new Error(
        `Channel ${mapped.name} holds ${dataType} data, not ${ch.dataType}. Set the ` +
          `data type to ${dataType}, or delete channel ${mapped.name} so the task can ` +
          "create it again.",
      );
  }
  return toCreate;
};
