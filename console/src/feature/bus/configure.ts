// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type bus,
  type channel,
  type library,
  type rack,
  type Synnax,
} from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import {
  commandChannelName,
  commandIndexName,
  fieldChannelName,
  indexName,
  rawName,
} from "@/feature/bus/names";
import { type Message, messagesOf } from "@/feature/bus/types";

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

const newIndex = (name: string): channel.New => ({
  name,
  dataType: DataType.TIMESTAMP,
  isIndex: true,
});

/**
 * Creates the channels in one call, reusing any that already exist by name.
 * @returns a function that gets the key of a channel by its name.
 * @throws {Error} from the returned function if no channel has the name.
 */
const createChannels = async (
  client: Synnax,
  channels: channel.New[],
): Promise<(name: string) => channel.Key> => {
  const byName = new Map(channels.map((c) => [c.name, c]));
  const created =
    byName.size === 0
      ? []
      : await client.channels.create([...byName.values()], {
          retrieveIfNameExists: true,
        });
  const keys = new Map(created.map((c) => [c.name, c.key]));
  return (name) => {
    const key = keys.get(name);
    if (key == null) throw new Error(`Channel ${name} was not created`);
    return key;
  };
};

/** A field to bind to a new channel named name on index. */
interface Unbound {
  field: { channel: channel.Key };
  name: string;
  index: channel.Key;
}

/** Binds each unbound field to a channel created in one call. */
const bindFields = async (client: Synnax, unbound: Unbound[]): Promise<void> => {
  const keyOf = await createChannels(
    client,
    unbound.map(({ name, index }) => ({ name, dataType: FIELD_DATA_TYPE, index })),
  );
  unbound.forEach(({ field, name }) => {
    field.channel = keyOf(name);
  });
};

interface Context {
  device: string;
  rack: rack.Key;
  messages: Map<library.EntryKey, library.MessageEntry>;
  existing: Set<channel.Key>;
}

const openContext = async (
  client: Synnax,
  config: bus.ReadConfig | bus.WriteConfig,
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

/**
 * @returns the name of the message field with the key.
 * @throws {Error} if the message has no such field.
 */
const fieldName = (entry: library.MessageEntry, key: library.FieldKey): string => {
  const field = entry.fields.find((f) => f.key === key);
  if (field == null) throw new Error(`Field ${key} is not in message ${entry.name}`);
  return field.name;
};

/**
 * Creates the channels a read config names but that do not exist: one index per
 * message, one channel per field, and the virtual raw frame channel. Channels that
 * already exist by name are reused.
 * @returns the config bound to its channels and the rack of its device.
 */
export const configureRead = async <C extends bus.ReadConfig>(
  client: Synnax,
  config: C,
): Promise<[C, rack.Key]> => {
  const keys = [
    config.raw,
    ...config.messages.flatMap((m) => [m.index, ...m.fields.map((f) => f.channel)]),
  ];
  const ctx = await openContext(client, config, keys);
  const enabled = config.messages
    .filter((m) => !m.disabled)
    .map((m) => ({ m, entry: entryOf(ctx, m) }));
  const raw = rawName(ctx.device);
  const rawMissing = !ctx.existing.has(config.raw);
  const unindexed = enabled.filter(({ m }) => !ctx.existing.has(m.index));
  const keyOf = await createChannels(client, [
    ...(rawMissing ? [{ name: raw, dataType: DataType.BYTES, virtual: true }] : []),
    ...unindexed.map(({ entry }) => newIndex(indexName(ctx.device, entry.name))),
  ]);
  if (rawMissing) config.raw = keyOf(raw);
  unindexed.forEach(({ m, entry }) => {
    m.index = keyOf(indexName(ctx.device, entry.name));
  });
  await bindFields(
    client,
    enabled.flatMap(({ m, entry }) =>
      m.fields
        .filter((f) => !ctx.existing.has(f.channel))
        .map((field) => ({
          field,
          name: fieldChannelName(ctx.device, entry.name, fieldName(entry, field.field)),
          index: m.index,
        })),
    ),
  );
  return [config, ctx.rack];
};

/**
 * Creates a command channel for each field a write config sends without one, on one
 * command index per message. Channels that already exist by name are reused.
 * @returns the config bound to its channels and the rack of its device.
 */
export const configureWrite = async <C extends bus.WriteConfig>(
  client: Synnax,
  config: C,
): Promise<[C, rack.Key]> => {
  const keys = config.messages.flatMap((m) => m.fields.map((f) => f.channel));
  const ctx = await openContext(client, config, keys);
  const pending = config.messages.flatMap((m) => {
    if (m.disabled) return [];
    const fields = m.fields.filter((f) => !ctx.existing.has(f.channel));
    return fields.length === 0 ? [] : [{ entry: entryOf(ctx, m), fields }];
  });
  const indexOf = await createChannels(
    client,
    pending.map(({ entry }) => newIndex(commandIndexName(ctx.device, entry.name))),
  );
  await bindFields(
    client,
    pending.flatMap(({ entry, fields }) => {
      const index = indexOf(commandIndexName(ctx.device, entry.name));
      return fields.map((field) => ({
        field,
        name: commandChannelName(ctx.device, entry.name, fieldName(entry, field.field)),
        index,
      }));
    }),
  );
  return [config, ctx.rack];
};
