// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  type device,
  ethercat,
  NotFoundError,
  type Synnax,
} from "@synnaxlabs/client";
import { DataType, errors, primitive } from "@synnaxlabs/x";

import {
  check,
  fill,
  findInput,
  sameAddress,
} from "@/feature/ethercat/device/settings";
import { canSwitch, isShared, of, onlyReaderOf } from "@/feature/ethercat/device/setup";
import {
  MAKE,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";
import {
  addressKey,
  DIRECTIONS,
  entryAt,
  type EntryChannel,
  type EntryReadChannel,
  getUnresolvedMessage,
  hasEntry,
  mappedChannels,
  type PDOType,
  type ReadChannel,
  usableEntries,
  type WriteChannel,
} from "@/feature/ethercat/task/types";
import { Device as PlatformDevice } from "@/platform/device";

type Channel = ReadChannel | WriteChannel;

export interface SlaveValidationResult<C extends Channel> {
  slaves: SlaveDevice[];
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

/** @returns what the user must do before Synnax can read an entry of the slave. */
const onlyReaderStep = (props: SlaveProperties): string => {
  if (of(props).kind === "switched") return "Restart the power of the module.";
  return canSwitch(props)
    ? "In Configure, switch the module to Synnax."
    : "Pick another entry.";
};

const validateEntries = <C extends Channel>(
  slave: SlaveDevice,
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
          `A channel on module ${slave.name} has no entry. Pick an entry.`,
        );
      const address = ethercat.formatAddress(ch);
      const entry = entryAt(entries, ch);
      if (entry == null)
        throw new Error(
          `Module ${slave.name} has no ${DIRECTIONS[pdoType]} entry at ${address}. ` +
            "Pick an entry.",
        );
      const reader = pdoType === "inputs" ? onlyReaderOf(slave.properties, ch) : null;
      if (reader != null)
        throw new Error(
          `Entry ${address} on module ${slave.name} sends data that only ${reader} ` +
            `reads. ${onlyReaderStep(slave.properties)}`,
        );
      const dataType = new DataType(ch.dataType);
      const width = entryWidth(entry.bitLength);
      if (
        !(dataType.isInteger || dataType.isFloat) ||
        dataType.density.valueOf() !== width
      )
        throw new Error(
          `Data type ${ch.dataType} does not fit entry ${address} on module ` +
            `${slave.name}, which holds ${entry.bitLength} bits. Pick a ${width} ` +
            "byte number type.",
        );
      const key = addressKey(ch);
      if (used.has(key))
        throw new Error(
          `More than one channel uses entry ${address} on module ${slave.name}. ` +
            "Pick a different entry for one of them.",
        );
      used.add(key);
      return ch;
    });
};

/**
 * @returns a channel from create for each usable entry on the side of the slaves with
 * the keys, in bus order, or the empty channel on the first slave when they have none.
 * When the catalog of a slave describes some of its entries, only those can get a
 * channel, so status entries stay out of the task.
 */
export const createSlaveChannels = async <C extends Channel>(
  client: Synnax,
  keys: device.Key[],
  pdoType: PDOType,
  create: (slave: SlaveDevice, entry: ethercat.PDOEntry) => C,
  empty: C,
): Promise<C[]> => {
  const slaves = await client.devices.retrieve({ key: keys, schemas: SLAVE_SCHEMAS });
  const channels = slaves
    .sort((a, b) => a.properties.position - b.properties.position)
    .flatMap((slave) => {
      const { properties } = slave;
      const described = (entry: ethercat.PDOEntry) =>
        findInput(properties.settings, entry) != null;
      const usable = usableEntries(properties, pdoType);
      const entries = properties.pdos[pdoType].some(described)
        ? usable.filter(described)
        : usable;
      return entries.map((entry) => create(slave, entry));
    });
  return channels.length === 0 ? [{ ...empty, device: keys[0] }] : channels;
};

/**
 * Retrieves the slaves the channels use and checks the entries of the enabled channels.
 * @returns the slaves, their rack, and the enabled channels of each slave.
 * @throws {Error} if the slaves are on different networks, or if an enabled channel has
 * no entry, an entry its slave lacks, an input that only another program reads, or an
 * entry another channel uses.
 */
export const retrieveAndValidateSlaves = async <C extends Channel>(
  client: Synnax,
  channels: C[],
  pdoType: PDOType,
): Promise<SlaveValidationResult<C>> => {
  const slaveKeys = [...new Set(channels.map((ch) => ch.device))];
  const slaves = await client.devices.retrieve({
    key: slaveKeys,
    schemas: SLAVE_SCHEMAS,
  });

  PlatformDevice.checkFound(slaveKeys, slaves);
  for (const slave of slaves) PlatformDevice.checkConfigured(slave);

  const networks = [...new Set(slaves.map((s) => s.properties.network))];
  if (networks.length > 1)
    throw new Error(
      `All modules must be on the same network. Found: ${networks.join(", ")}. ` +
        "Pick modules on one network.",
    );
  if (networks.length === 0 || !networks[0])
    throw new Error(
      "No network found for the selected modules. Scan the network again.",
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

/**
 * @returns the startup parameters of each slave: a value for every field of each input
 * that an enabled channel reads, in the order of the slave's catalog. Values the task
 * held stay. A slave that other software sets up gets none.
 * @throws {Error} if a value breaks a limit or a rule of the catalog.
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
    for (const input of inputs) {
      const ch = entries.find((e) => input.entries.some((a) => sameAddress(a, e)));
      if (ch == null) continue;
      const filledInput = fill(input.settings, prior);
      const [problem] = check(input, filledInput);
      if (problem != null)
        throw new Error(
          `Entry ${ethercat.formatAddress(ch)} on module ${slave.name}: ${problem.message}`,
        );
      parameters.push(...filledInput);
    }
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
      `Module ${owner.name} uses the index channel ${index.name}. Change the ` +
        `identifier of module ${slave.name}.`,
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
            `identifier of module ${slave.name}.`,
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
