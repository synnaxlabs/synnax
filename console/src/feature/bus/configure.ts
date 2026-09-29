// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type library, type rack, type Synnax } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import {
  commandChannelName,
  commandIndexName,
  fieldChannelName,
  indexName,
  rawName,
} from "@/feature/bus/names";
import {
  type Message,
  messagesOf,
  type ReadConfig,
  type WriteConfig,
} from "@/feature/bus/types";

const FIELD_DATA_TYPE = DataType.FLOAT64.toString();

/** Retrieves the keys among keys whose channels exist. */
const retrieveExisting = async (
  client: Synnax,
  keys: channel.Key[],
): Promise<Set<channel.Key>> => {
  const nonZero = keys.filter((k) => k !== 0);
  if (nonZero.length === 0) return new Set();
  const found = await client.channels.retrieve(nonZero);
  return new Set(found.map((c) => c.key));
};

const createIndex = async (client: Synnax, name: string): Promise<channel.Key> =>
  (
    await client.channels.create(
      { name, dataType: DataType.TIMESTAMP, isIndex: true },
      { retrieveIfNameExists: true },
    )
  ).key;

/**
 * Creates a channel for each name on index, reusing any that already exist by name.
 * @returns the key of each channel by name.
 */
const createFieldChannels = async (
  client: Synnax,
  names: string[],
  index: channel.Key,
): Promise<Map<string, channel.Key>> => {
  const created = await client.channels.create(
    names.map((name) => ({ name, dataType: FIELD_DATA_TYPE, index })),
    { retrieveIfNameExists: true },
  );
  return new Map(created.map((c) => [c.name, c.key]));
};

interface Context {
  device: string;
  rack: rack.Key;
  messages: Map<library.EntryKey, library.MessageEntry>;
  existing: Set<channel.Key>;
}

const openContext = async (
  client: Synnax,
  config: ReadConfig | WriteConfig,
  keys: channel.Key[],
): Promise<Context> => {
  const [dev, lib, existing] = await Promise.all([
    client.devices.retrieve({ key: config.device }),
    client.libraries.retrieve({ key: config.library }),
    retrieveExisting(client, keys),
  ]);
  return { device: dev.name, rack: dev.rack, messages: messagesOf(lib), existing };
};

const entryOf = (ctx: Context, m: Message): library.MessageEntry => {
  const entry = ctx.messages.get(m.message);
  if (entry == null) throw new Error(`Message ${m.message} is not in the library`);
  return entry;
};

/** @returns the name of a message field, throwing when the message has no such field. */
const fieldName = (entry: library.MessageEntry, key: library.FieldKey): string => {
  const field = entry.fields.find((f) => f.key === key);
  if (field == null) throw new Error(`Field ${key} is not in message ${entry.name}`);
  return field.name;
};

/**
 * Binds each unbound field to the channel createFieldChannels made for its name.
 * @param nameOf - The channel name of a field.
 */
const bindFields = async (
  client: Synnax,
  unbound: { field: library.FieldKey; channel: channel.Key }[],
  index: channel.Key,
  nameOf: (field: library.FieldKey) => string,
): Promise<void> => {
  const created = await createFieldChannels(
    client,
    unbound.map((f) => nameOf(f.field)),
    index,
  );
  unbound.forEach((f) => {
    const key = created.get(nameOf(f.field));
    if (key == null) throw new Error(`Channel ${nameOf(f.field)} was not created`);
    f.channel = key;
  });
};

/**
 * Creates the channels a read config names but that do not exist: one index per
 * message, one channel per field, and the virtual raw frame channel. Channels that
 * already exist by name are reused.
 * @returns the config bound to its channels and the rack of its device.
 */
export const configureRead = async <C extends ReadConfig>(
  client: Synnax,
  config: C,
): Promise<[C, rack.Key]> => {
  const keys = [
    config.raw,
    ...config.messages.flatMap((m) => [m.index, ...m.fields.map((f) => f.channel)]),
  ];
  const ctx = await openContext(client, config, keys);
  if (!ctx.existing.has(config.raw))
    config.raw = (
      await client.channels.create(
        { name: rawName(ctx.device), dataType: DataType.BYTES, virtual: true },
        { retrieveIfNameExists: true },
      )
    ).key;
  for (const m of config.messages) {
    if (m.disabled) continue;
    const entry = entryOf(ctx, m);
    if (!ctx.existing.has(m.index))
      m.index = await createIndex(client, indexName(ctx.device, entry.name));
    const unbound = m.fields.filter((f) => !ctx.existing.has(f.channel));
    if (unbound.length === 0) continue;
    await bindFields(client, unbound, m.index, (key) =>
      fieldChannelName(ctx.device, entry.name, fieldName(entry, key)),
    );
  }
  return [config, ctx.rack];
};

/**
 * Creates a command channel for each field a write config sends without one, on one
 * command index per message. Channels that already exist by name are reused.
 * @returns the config bound to its channels and the rack of its device.
 */
export const configureWrite = async <C extends WriteConfig>(
  client: Synnax,
  config: C,
): Promise<[C, rack.Key]> => {
  const keys = config.messages.flatMap((m) => m.fields.map((f) => f.channel));
  const ctx = await openContext(client, config, keys);
  for (const m of config.messages) {
    if (m.disabled) continue;
    const unbound = m.fields.filter((f) => !ctx.existing.has(f.channel));
    if (unbound.length === 0) continue;
    const entry = entryOf(ctx, m);
    const index = await createIndex(client, commandIndexName(ctx.device, entry.name));
    await bindFields(client, unbound, index, (key) =>
      commandChannelName(ctx.device, entry.name, fieldName(entry, key)),
    );
  }
  return [config, ctx.rack];
};
