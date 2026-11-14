// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type ethercat, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { errors, primitive } from "@synnaxlabs/x";

import { fill, findInput, sameAddress } from "@/feature/ethercat/device/settings";
import { isShared } from "@/feature/ethercat/device/setup";
import {
  MAKE,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
} from "@/feature/ethercat/device/types";
import {
  type EntryChannel,
  type EntryReadChannel,
  hasEntry,
  mappedChannels,
  type PDOType,
  type ReadChannel,
  usableEntries,
  type WriteChannel,
} from "@/feature/ethercat/task/types";
import { Device as PlatformDevice } from "@/platform/device";

type Channel = ReadChannel | WriteChannel;

export interface Slaves<C extends Channel> {
  slaves: SlaveDevice[];
  rack: number;
  channelsBySlaveKey: Map<string, Extract<C, EntryChannel>[]>;
}

/**
 * @returns a channel from create for each usable entry on the side of the slave with
 * the key, or the empty channel on the slave when it has none. When the catalog of the
 * slave describes some of its entries, only those can get a channel, so status entries
 * stay out of the task.
 */
export const createSlaveChannels = async <C extends Channel>(
  client: Synnax,
  key: string,
  pdoType: PDOType,
  create: (slave: SlaveDevice, entry: ethercat.PDOEntry) => C,
  empty: C,
): Promise<C[]> => {
  const slave = await client.devices.retrieve({ key, schemas: SLAVE_SCHEMAS });
  const { properties } = slave;
  const described = (entry: ethercat.PDOEntry) =>
    findInput(properties.settings, entry) != null;
  const usable = usableEntries(properties, pdoType);
  const entries = properties.pdos[pdoType].some(described)
    ? usable.filter(described)
    : usable;
  if (entries.length === 0) return [{ ...empty, device: key }];
  return entries.map((entry) => create(slave, entry));
};

/**
 * Retrieves the slaves the channels use. The form checks the slaves and channels before
 * a deploy reaches this.
 * @returns the slaves, their rack, and the enabled channels with an entry on each
 * slave.
 * @throws {NotFoundError} if a slave of a channel no longer exists.
 */
export const retrieveSlaves = async <C extends Channel>(
  client: Synnax,
  channels: C[],
): Promise<Slaves<C>> => {
  const slaveKeys = [...new Set(channels.map((ch) => ch.device))];
  const slaves = await client.devices.retrieve({
    key: slaveKeys,
    schemas: SLAVE_SCHEMAS,
  });
  PlatformDevice.checkFound(slaveKeys, slaves);
  for (const slave of slaves) PlatformDevice.checkConfigured(slave);
  const channelsBySlaveKey = new Map<string, Extract<C, EntryChannel>[]>();
  for (const slave of slaves)
    channelsBySlaveKey.set(
      slave.key,
      channels.filter(
        (ch): ch is Extract<C, EntryChannel> =>
          ch.device === slave.key && !ch.disabled && hasEntry(ch),
      ),
    );
  return { slaves, rack: slaves[0].rack, channelsBySlaveKey };
};

/**
 * @returns the startup parameters of each slave: a value for every field of each input
 * that an enabled channel reads, in the order of the slave's catalog. Values the task
 * held stay. A slave that other software sets up gets none.
 */
export const fillStartupParameters = (
  slaves: SlaveDevice[],
  channels: ReadChannel[],
  held: ethercat.SlaveStartupParameters[],
): ethercat.SlaveStartupParameters[] => {
  const filled: ethercat.SlaveStartupParameters[] = [];
  for (const slave of slaves) {
    if (isShared(slave.properties)) continue;
    const inputs = slave.properties.settings?.channels ?? [];
    const entries = channels.filter(
      (ch): ch is EntryReadChannel =>
        hasEntry(ch) && !ch.disabled && ch.device === slave.key,
    );
    const prior = held.find((s) => s.device === slave.key)?.parameters ?? [];
    const parameters: ethercat.StartupParameter[] = [];
    for (const input of inputs)
      if (entries.some((e) => input.entries.some((a) => sameAddress(a, e))))
        parameters.push(...fill(input.settings, prior));
    if (parameters.length > 0) filled.push({ device: slave.key, parameters });
  }
  return filled;
};

export interface IndexOptions {
  indexProperty: "readIndex" | "writeStateIndex";
  channelsProperty: "read" | "write";
  nameSuffix: "_time" | "_state_time";
}

/** @returns the channels that hold the names, keyed by name. */
export const retrieveByName = async (
  client: Synnax,
  names: string[],
): Promise<Map<string, channel.Channel>> => {
  if (names.length === 0) return new Map();
  const found = await client.channels.retrieve({ name: names });
  return new Map(found.map((c) => [c.name, c]));
};

/**
 * @throws {Error} if a device other than the slave uses the slave's identifier, or if
 * another slave uses the index for itself or for a command channel.
 */
const checkIndexFree = async (
  client: Synnax,
  slave: SlaveDevice,
  index: channel.Channel,
): Promise<void> => {
  const identifier = channel.escapeInvalidName(slave.properties.identifier);
  const other = (await client.devices.retrieve({})).find(
    ({ key, properties }) =>
      key !== slave.key &&
      typeof properties?.identifier === "string" &&
      channel.escapeInvalidName(properties.identifier) === identifier,
  );
  if (other != null)
    throw new Error(
      `Device ${other.name} has an identifier that gives the same channel names as ` +
        `${slave.properties.identifier}. Change the identifier of one of them.`,
    );
  const others = (
    await client.devices.retrieve({
      make: MAKE,
      model: SLAVE_MODEL,
      schemas: SLAVE_SCHEMAS,
    })
  ).filter(({ key }) => key !== slave.key);
  const commands = others.flatMap(({ properties }) =>
    properties.write.channels.map(({ command }) => command).filter((key) => key !== 0),
  );
  const indexes = new Map(
    commands.length === 0
      ? []
      : (await client.channels.retrieve({ key: commands })).map((ch) => [
          ch.key,
          ch.index,
        ]),
  );
  const owner = others.find(
    ({ properties: { readIndex, writeStateIndex, write } }) =>
      readIndex === index.key ||
      writeStateIndex === index.key ||
      write.channels.some(({ command }) => indexes.get(command) === index.key),
  );
  if (owner != null)
    throw new Error(
      `Slave ${owner.name} uses the index channel ${index.name}. Change the ` +
        `identifier of slave ${slave.name}.`,
    );
};

/**
 * Binds the slave to its index channel, creating the channel when the slave has none.
 * A deleted device leaves its channels, so the slave reuses an index channel that
 * holds its name when no other device uses its identifier and no other slave uses the
 * index.
 *
 * @returns true when the slave's properties changed.
 * @throws {Error} if a channel that is not an index holds the index name, if another
 * device uses the slave's identifier, or if another slave uses the index.
 */
export const checkOrCreateIndex = async (
  client: Synnax,
  slave: SlaveDevice,
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
    const name = `${identifier}${nameSuffix}`;
    let idx = (await retrieveByName(client, [name])).get(name);
    if (idx != null) {
      if (!idx.isIndex)
        throw new Error(
          `A channel named ${name} already exists and is not an index. Change the ` +
            `identifier of slave ${slave.name}.`,
        );
      await checkIndexFree(client, slave, idx);
    } else
      idx = await client.channels.create({
        name,
        dataType: "timestamp",
        isIndex: true,
      });
    if (indexProperty === "readIndex") slave.properties.readIndex = idx.key;
    else slave.properties.writeStateIndex = idx.key;
    slave.properties[channelsProperty].channels = [];
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
  channelOf: (ch: C) => number,
): Promise<C[]> => {
  const toCreate: C[] = [];
  for (const ch of channels) {
    const key = channelOf(ch);
    if (key === 0) {
      toCreate.push(ch);
      continue;
    }
    let existing: channel.Channel;
    try {
      existing = await client.channels.retrieve(key);
    } catch (e) {
      if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      toCreate.push(ch);
      continue;
    }
    checkDataType(existing, ch.dataType);
  }
  return toCreate;
};

/** @throws {Error} if the channel holds a data type other than the one given. */
const checkDataType = (existing: channel.Channel, dataType: string): void => {
  const held = existing.dataType.toString();
  if (held !== dataType)
    throw new Error(
      `Channel ${existing.name} holds ${held} data, not ${dataType}. Set the data ` +
        `type to ${held}, or delete channel ${existing.name} so the task can create ` +
        "it again.",
    );
};

export interface BindOrphanedChannelsArgs<C extends EntryChannel> {
  client: Synnax;
  /**
   * the key of the slave that deploys the channels. Set it when the channels do not use
   * the slave's index, to reject a channel that another slave maps.
   */
  slave?: string;
  /** the channels that have no Synnax channel yet. */
  channels: C[];
  /** the name deploying gives each channel's Synnax channel. */
  names: string[];
  /**
   * @returns the index the Synnax channel of the channel at i must have, or undefined
   * when the channel must not reuse a Synnax channel.
   */
  index: (i: number) => number | undefined;
  /** the Synnax channels that the device maps, which no channel may reuse. */
  mapped: number[];
  /** maps the channel to the Synnax channel with the key on the device. */
  bind: (ch: C, key: number) => void;
  /** @returns the data type the Synnax channel of the channel must hold. */
  getDataType?: (ch: C) => string;
}

/**
 * Binds each channel to the Synnax channel that a deleted device left under its name.
 * An orphaned channel is not an index, has the index the task gives the channel, and no
 * slave maps it.
 *
 * @returns the channels that still need a Synnax channel.
 * @throws {Error} if an index, a channel on another index, or a channel that a slave
 * maps holds a name, or if an orphaned channel holds another data type.
 */
export const bindOrphanedChannels = async <C extends EntryChannel>({
  client,
  slave,
  channels,
  names,
  index,
  mapped,
  bind,
  getDataType = (ch) => ch.dataType,
}: BindOrphanedChannelsArgs<C>): Promise<C[]> => {
  const existing = await retrieveByName(client, names);
  if (existing.size === 0) return channels;
  const others =
    slave == null
      ? []
      : await client.devices.retrieve({
          make: MAKE,
          model: SLAVE_MODEL,
          schemas: SLAVE_SCHEMAS,
        });
  const bound = new Set([
    ...mapped,
    ...others.filter(({ key }) => key !== slave).flatMap(mappedChannels),
  ]);
  return channels.filter((ch, i) => {
    const orphan = existing.get(names[i]);
    if (orphan == null) return true;
    const expected = index(i);
    if (
      expected == null ||
      orphan.isIndex ||
      orphan.index !== expected ||
      bound.has(orphan.key)
    )
      throw new Error(
        `A channel named ${orphan.name} already exists. Give the channel another name.`,
      );
    checkDataType(orphan, getDataType(ch));
    bind(ch, orphan.key);
    bound.add(orphan.key);
    return false;
  });
};
