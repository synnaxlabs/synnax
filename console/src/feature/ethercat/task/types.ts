// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, ethercat, type task } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { z } from "zod";

import { type Device } from "@/feature/ethercat/device";
import { Task } from "@/platform/task";

export const PREFIX = "ethercat";

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

export interface Address {
  index: number;
  subIndex: number;
}

const ZERO_ENTRY = { index: 0, subIndex: 0, dataType: "" } as const;

/** @returns whether the channel has an entry picked. */
export const hasEntry = <C extends Channel>(ch: C): ch is Extract<C, EntryChannel> =>
  ch.type === "entry" && ch.index !== 0;

/** The word for the direction of each side of the process image. */
export const DIRECTIONS: Record<PDOType, string> = {
  inputs: "input",
  outputs: "output",
};

/**
 * @returns the key that maps the entry at the address to its Synnax channel in the
 * slave's device properties. The key survives the wire's case conversion unchanged.
 */
export const addressKey = ({ index, subIndex }: Address): string =>
  `${index}_${subIndex}`;

/** @returns the device properties key of the write state channel at the address. */
export const stateKey = (address: Address): string => `state_${addressKey(address)}`;

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
  { index, subIndex }: Address,
): ethercat.PDOEntry | undefined =>
  entries.find((e) => e.index === index && e.subIndex === subIndex);

/** @returns the entry label, e.g. "Channel 2 › Input", or the name alone. */
export const getEntryLabel = ({ name, pdoName }: ethercat.PDOEntry): string =>
  pdoName == null || pdoName === "" ? name : `${pdoName} › ${name}`;

/**
 * @returns the error for a channel whose entry name matches no single entry, listing
 * the entries on the slave that carry the name.
 */
export const getUnresolvedMessage = (
  pdo: string,
  { name, properties }: Pick<Device.SlaveDevice, "name" | "properties">,
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

/** Creates a new write channel, copying the slave of the last channel if available. */
export const createWriteChannel = (channels: WriteChannel[]): WriteChannel => {
  const base = channels.length === 0 ? {} : channels[channels.length - 1];
  return WRITE_CHANNEL_SCHEMAS.entry.parse({
    ...base,
    ...Task.WRITE_CHANNEL_OVERRIDE,
    ...ZERO_ENTRY,
    type: "entry",
    name: "",
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
