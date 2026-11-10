// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, ethercat, type task } from "@synnaxlabs/client";
import { id, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Address,
  type SlaveDevice,
  type WriteBinding,
} from "@/feature/ethercat/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "ethercat";

export const SCAN_TYPE = `${PREFIX}_scan`;

export type ReadChannel = ethercat.ReadChannel;

export const READ_CHANNEL_SCHEMAS = ethercat.READ_CHANNEL_SCHEMAS;

export type WriteChannel = ethercat.WriteChannel;

export const WRITE_CHANNEL_SCHEMAS = ethercat.WRITE_CHANNEL_SCHEMAS;

export type Channel = ReadChannel | WriteChannel;

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = ethercat.readConfigZ;

export const deployReadConfigZ = ethercat.readConfigZ
  .extend({
    channels: ethercat.readConfigZ.shape.channels.check(Task.validateEnabledChannel),
    sampleRate: z.number().positive(),
    streamRate: z.number().positive(),
  })
  .check(Task.validateStreamRate);

const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = ethercat.writeConfigZ;

export const deployWriteConfigZ = ethercat.writeConfigZ.extend({
  channels: ethercat.writeConfigZ.shape.channels.check(Task.validateEnabledChannel),
  stateRate: z.number().positive(),
  executionRate: z.number().positive(),
});

const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: writeStatusDataZ,
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export type EntryReadChannel = ethercat.EntryReadChannel;

export type EntryWriteChannel = ethercat.EntryWriteChannel;

export type EntryChannel = EntryReadChannel | EntryWriteChannel;

/**
 * The side of the process image a task reaches: inputs for reads, outputs for writes.
 */
export type PDOType = "inputs" | "outputs";

const ZERO_ENTRY = { index: 0, subIndex: 0, dataType: "" } as const;

/** @returns whether the channel has an entry picked. */
export const hasEntry = <C extends Channel>(ch: C): ch is Extract<C, EntryChannel> =>
  ch.type === "entry" && ch.index !== 0;

/** The word for the direction of each side of the process image. */
export const DIRECTIONS: Record<PDOType, string> = {
  inputs: "input",
  outputs: "output",
};

/** @returns a key that is unique to the address among the entries of a slave. */
export const addressKey = ({ index, subIndex }: Address): string =>
  `${index}_${subIndex}`;

const isAt =
  ({ index, subIndex }: Address) =>
  (b: Address): boolean =>
    b.index === index && b.subIndex === subIndex;

/** @returns whether the read channel scales its samples. */
const isScaled = (ch: EntryReadChannel): boolean => ch.scale.type !== "none";

/** @returns the data type of the read channel's Synnax channel. */
export const readDataType = (ch: EntryReadChannel): string =>
  isScaled(ch) ? "float64" : ch.dataType;

/**
 * @returns the Synnax channel that the slave maps for the read channel, or 0. A scaled
 * channel writes float64 samples, so it maps to its own Synnax channel.
 */
export const readChannelOf = (slave: SlaveDevice, ch: EntryReadChannel): number => {
  const at = isAt(ch);
  const scaled = isScaled(ch);
  return (
    slave.properties.read.channels.find((b) => at(b) && b.scaled === scaled)?.channel ??
    0
  );
};

/** Maps the read channel to the Synnax channel with the key on the slave. */
export const bindReadChannel = (
  slave: SlaveDevice,
  ch: EntryReadChannel,
  key: number,
): void => {
  const at = isAt(ch);
  const scaled = isScaled(ch);
  const { channels } = slave.properties.read;
  const binding = channels.find((b) => at(b) && b.scaled === scaled);
  if (binding != null) binding.channel = key;
  else channels.push({ index: ch.index, subIndex: ch.subIndex, scaled, channel: key });
};

/** The Synnax channel of an output entry: its command or its state. */
type WriteRole = keyof Pick<WriteBinding, "command" | "state">;

/** @returns the channel that the slave maps for the role at the address, or 0. */
export const writeChannelOf = (
  slave: SlaveDevice,
  address: Address,
  role: WriteRole,
): number => slave.properties.write.channels.find(isAt(address))?.[role] ?? 0;

/** Maps the role at the address to the Synnax channel with the key on the slave. */
export const bindWriteChannel = (
  slave: SlaveDevice,
  address: Address,
  role: WriteRole,
  key: number,
): void => {
  const { channels } = slave.properties.write;
  let binding = channels.find(isAt(address));
  if (binding == null) {
    binding = {
      index: address.index,
      subIndex: address.subIndex,
      command: 0,
      state: 0,
    };
    channels.push(binding);
  }
  binding[role] = key;
};

/** @returns every Synnax channel that the slave maps. */
export const mappedChannels = ({
  properties: { read, write },
}: SlaveDevice): number[] =>
  [
    ...read.channels.map(({ channel }) => channel),
    ...write.channels.flatMap(({ command, state }) => [command, state]),
  ].filter((key) => key !== 0);

/** @returns the entries of a slave with each address listed once. */
export const uniqueEntries = (entries: ethercat.PDOEntry[]): ethercat.PDOEntry[] => {
  const seen = new Set<string>();
  return entries.filter((entry) => {
    const key = addressKey(entry);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
};

/** @returns the entry at the address, or undefined if the slave has none there. */
export const entryAt = (
  entries: ethercat.PDOEntry[],
  address: Address,
): ethercat.PDOEntry | undefined => entries.find(isAt(address));

/** @returns the entry label, e.g. "Channel 2 › Input", or the name alone. */
export const getEntryLabel = ({ name, pdoName }: ethercat.PDOEntry): string =>
  pdoName == null || pdoName === "" ? name : `${pdoName} › ${name}`;

/**
 * @returns the error for a channel whose entry name matches no single entry, listing
 * the entries on the slave that carry the name.
 */
export const getUnresolvedMessage = (
  pdo: string,
  { name, properties }: Pick<SlaveDevice, "name" | "properties">,
  pdoType: PDOType,
): string => {
  const direction = DIRECTIONS[pdoType];
  const matches = uniqueEntries(properties.pdos[pdoType])
    .filter((e) => e.name === pdo)
    .map(ethercat.formatEntry);
  const slave = `on slave ${name}`;
  let found: string;
  if (matches.length === 0) found = `no ${direction} entry ${slave}`;
  else if (matches.length === 1) found = `${direction} entry ${matches[0]} ${slave}`;
  else found = `more than one ${direction} entry ${slave}: ${matches.join(", ")}`;
  return `'${pdo}' matches ${found}. Pick an entry.`;
};

/** Creates a new read channel, copying the slave of the last channel if available. */
export const createReadChannel = (channels: ReadChannel[]): ReadChannel => {
  const base = channels.length === 0 ? {} : channels[channels.length - 1];
  return READ_CHANNEL_SCHEMAS.entry.parse({
    ...base,
    ...Task.READ_CHANNEL_OVERRIDE,
    ...ZERO_ENTRY,
    type: "entry",
    key: id.create(),
  });
};

/** @returns a read channel bound to an entry of the slave with the given key. */
export const createEntryReadChannel = (
  device: string,
  { index, subIndex, dataType }: ethercat.PDOEntry,
): EntryReadChannel =>
  READ_CHANNEL_SCHEMAS.entry.parse({
    type: "entry",
    device,
    index,
    subIndex,
    dataType,
  });

/** @returns a write channel bound to an entry of the slave with the given key. */
export const createEntryWriteChannel = (
  device: string,
  { index, subIndex, dataType }: ethercat.PDOEntry,
): EntryWriteChannel =>
  WRITE_CHANNEL_SCHEMAS.entry.parse({
    type: "entry",
    device,
    index,
    subIndex,
    dataType,
  });

/** Creates a new write channel, copying the slave of the last channel if available. */
export const createWriteChannel = (channels: WriteChannel[]): WriteChannel => {
  const base = channels.length === 0 ? {} : channels[channels.length - 1];
  return WRITE_CHANNEL_SCHEMAS.entry.parse({
    ...base,
    ...Task.WRITE_CHANNEL_OVERRIDE,
    ...ZERO_ENTRY,
    type: "entry",
    name: "",
    stopValue: 0,
    key: id.create(),
  });
};

/** @returns the short label of a channel's entry for lists and titles. */
export const getPortLabel = (ch: Channel): string => {
  if (ch.type === "unresolved") return ch.pdo;
  if (!hasEntry(ch)) return "No entry";
  return ethercat.formatAddress(ch);
};

/**
 * @returns a valid Synnax channel name for the entry at the address on a slave. The
 * entry name alone is used when it is unique among the slave's entries; otherwise the
 * PDO name, or the address when the slave has no PDO name, tells it apart.
 * @throws {Error} if the slave has no entry at the address.
 */
export const getEntryChannelName = (
  identifier: string,
  entries: ethercat.PDOEntry[],
  address: Address,
): string => {
  const entry = entryAt(entries, address);
  if (entry == null)
    throw new Error(`The slave has no entry at ${ethercat.formatAddress(address)}`);
  let part: string;
  if (uniqueEntries(entries).filter((e) => e.name === entry.name).length === 1)
    part = entry.name;
  else if (entry.pdoName != null && entry.pdoName !== "")
    part = `${entry.pdoName}_${entry.name}`;
  else {
    const hex = ethercat.formatAddress(address).slice(2).replace(":", "_");
    part = `${entry.name}_${hex}`;
  }
  return channel.escapeInvalidName(`${identifier}_${part}`);
};

/**
 * The name deploying the task gives a read channel. The default name of a scaled
 * channel ends in `_scaled`, apart from the raw channel of its entry.
 */
export const readChannelName = (slave: SlaveDevice, ch: EntryReadChannel): string => {
  if (primitive.isNonZero(ch.name)) return ch.name;
  const name = getEntryChannelName(
    slave.properties.identifier,
    slave.properties.pdos.inputs,
    ch,
  );
  return isScaled(ch) ? `${name}_scaled` : name;
};

/** The names deploying the task gives a write channel's command and state channels. */
export const writeChannelNames = (
  slave: SlaveDevice,
  ch: EntryWriteChannel,
): { cmd: string; state: string } => {
  const base = getEntryChannelName(
    slave.properties.identifier,
    slave.properties.pdos.outputs,
    ch,
  );
  return {
    cmd: primitive.isNonZero(ch.cmdChannelName) ? ch.cmdChannelName : `${base}_cmd`,
    state: primitive.isNonZero(ch.stateChannelName)
      ? ch.stateChannelName
      : `${base}_state`,
  };
};
