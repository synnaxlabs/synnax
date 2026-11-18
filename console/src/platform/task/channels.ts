// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax as Client } from "@synnaxlabs/client";
import { DataType, errors, primitive } from "@synnaxlabs/x";

/** @returns the channel with the given key, or null when it does not exist or is 0. */
const retrieveChannel = async (
  client: Client,
  key: channel.Key,
): Promise<channel.Channel | null> => {
  if (!primitive.isNonZero(key)) return null;
  try {
    return await client.channels.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

/** @returns true when a channel with the given key exists. Never for 0. */
const channelExists = async (client: Client, key: channel.Key): Promise<boolean> =>
  (await retrieveChannel(client, key)) != null;

/**
 * Creates a Core channel for values of dataType. A fixed-density channel gets its own
 * index named `${name}_time`; a variable-density one is virtual.
 */
const createChannelWithIndex = async (
  client: Client,
  name: string,
  dataType: string,
): Promise<channel.Channel> => {
  if (new DataType(dataType).isVariable)
    return await client.channels.create({ name, dataType, virtual: true });
  const index = await client.channels.create({
    name: `${name}_time`,
    dataType: "timestamp",
    isIndex: true,
  });
  return await client.channels.create({ name, dataType, index: index.key });
};

/** The channels of one read source, as the properties of a device store them. */
export interface ReadChannelProps {
  index: channel.Key;
  channels: Record<string, channel.Key>;
}

/** A field of a read source that gets a channel. */
export interface ReadChannelField {
  key: string;
  pointer: string;
  channel: channel.Key;
  name: string;
  dataType: string;
  /** A disabled field gets no channel. */
  disabled?: boolean;
  /** The channel that a new channel of the field replaces, as it no longer suits it. */
  replaces?: channel.Key;
}

/** Where a device stores the channels of one read source. */
export interface StoredChannels {
  /** The read properties of the device. Updated in place. */
  read: Record<string, ReadChannelProps>;
  /** The key of the source in read. */
  key: string;
}

export interface ConfigureReadChannelsArgs<F extends ReadChannelField> {
  client: Client;
  /** Where the device stores the channels of the source. Absent when it stores none. */
  stored?: StoredChannels;
  /** The fields of the source. Each enabled one gets its channel in place. */
  fields: F[];
  /** Prefix of a generated channel name. A new index is named `${namePrefix}_time`. */
  namePrefix: string;
  /** Key of the field whose channel is the index. */
  indexKey: string;
  /**
   * Whether a field stores its samples on the index of the source. Defaults to every
   * fixed-density field.
   */
  indexed?: (field: F) => boolean;
}

const isFixed = (field: ReadChannelField): boolean =>
  !new DataType(field.dataType).isVariable;

/**
 * @returns each of names, or the first of `${name}_2`, `${name}_3`, and so on that no
 * channel on the Core and no earlier name holds.
 */
const freeNames = async (client: Client, names: string[]): Promise<string[]> => {
  const taken = new Set(
    (await client.channels.retrieve(names)).map(({ name }) => name),
  );
  const free: string[] = [];
  for (const name of names) {
    let candidate = name;
    for (let n = 2; taken.has(candidate); n++) {
      candidate = `${name}_${n}`;
      if ((await client.channels.retrieve([candidate])).length > 0)
        taken.add(candidate);
    }
    taken.add(candidate);
    free.push(candidate);
  }
  return free;
};

/**
 * Gives every enabled field of a read source a channel: the one it has, the one the
 * device stores for its pointer, or a new one, which it adds to stored. The index field
 * or a new indexed channel gets the stored index. When that is gone, it gets the index
 * of a live channel of the source, or a new one. A source that needs no new channel
 * gets no entry in stored. A new channel whose name is taken gets a number, such as
 * `_2`, and fields claim their names before a new index. A replaced channel that holds
 * the name of its replacement moves to a numbered name first, so the replacement keeps
 * the plain one.
 */
export const configureReadChannels = async <F extends ReadChannelField>({
  client,
  stored,
  fields,
  namePrefix,
  indexKey,
  indexed = isFixed,
}: ConfigureReadChannelsArgs<F>): Promise<void> => {
  const props = stored?.read[stored.key] ?? { index: 0, channels: {} };
  const save = () => {
    if (stored != null) stored.read[stored.key] = props;
  };
  const enabled = fields.filter(({ disabled }) => !disabled);
  const live: channel.Channel[] = [];
  const bind = async (field: F): Promise<boolean> => {
    if (primitive.isNonZero(field.channel)) {
      if (indexed(field)) live.push(await client.channels.retrieve(field.channel));
      return true;
    }
    const ch = props.channels[field.pointer];
    if (!(await channelExists(client, ch))) return false;
    field.channel = ch;
    return true;
  };
  const indexField = enabled.find(({ key }) => key === indexKey);
  const unbound: F[] = [];
  for (const field of enabled)
    if (field !== indexField && !(await bind(field))) unbound.push(field);
  const needsIndex =
    enabled.some(indexed) && (indexField != null || unbound.some(indexed));
  let createsIndex = false;
  if (needsIndex && !(await channelExists(client, props.index))) {
    props.index = await recoverIndex(client, live, props);
    createsIndex = !primitive.isNonZero(props.index);
    save();
  }
  const takesIndex = createsIndex || primitive.isNonZero(props.index);
  if (indexField != null && !takesIndex && !(await bind(indexField)))
    unbound.push(indexField);
  const planned = unbound.map((field) =>
    primitive.isNonZero(field.name)
      ? field.name
      : `${namePrefix}${channel.escapeInvalidName(field.pointer)}`,
  );
  for (const [i, field] of unbound.entries()) {
    const old = await retrieveChannel(client, field.replaces ?? 0);
    if (old == null || old.name !== planned[i]) continue;
    const [numbered] = await freeNames(client, [old.name]);
    await client.channels.rename(old.key, numbered);
  }
  const names = await freeNames(client, [
    ...planned,
    ...(createsIndex ? [`${namePrefix}_time`] : []),
  ]);
  if (createsIndex) {
    const indexCh = await client.channels.create({
      name: names[unbound.length],
      dataType: "timestamp",
      isIndex: true,
    });
    props.index = indexCh.key;
  }
  if (indexField != null && takesIndex) indexField.channel = props.index;
  for (const [i, field] of unbound.entries()) {
    const ch = await client.channels.create({
      name: names[i],
      dataType: field.dataType,
      ...(indexed(field) ? { index: props.index } : { virtual: true }),
    });
    field.channel = ch.key;
    props.channels[field.pointer] = ch.key;
    save();
  }
};

/**
 * @returns the first index among those of the live channels, then those of the stored
 * channels that still exist, or 0.
 */
const recoverIndex = async (
  client: Client,
  live: channel.Channel[],
  props: ReadChannelProps,
): Promise<channel.Key> => {
  for (const ch of live) if (primitive.isNonZero(ch.index)) return ch.index;
  for (const stored of Object.values(props.channels)) {
    const ch = await retrieveChannel(client, stored);
    if (ch != null && primitive.isNonZero(ch.index)) return ch.index;
  }
  return 0;
};

/** What a write target needs from its command channel. */
export interface CommandChannelSpec {
  /** The key of the target in the write properties of the device. */
  propertiesKey: string;
  /** The channel the target has now, or 0. */
  channel: channel.Key;
  /** The name of a channel to create. */
  name: string;
  dataType: string;
}

/**
 * Gives a write target a command channel: the one it has, the one the device stores
 * for its key, or a new one.
 *
 * @param write - The write properties of the device. Updated in place.
 * @returns the command channel.
 */
export const configureCommandChannel = async (
  client: Client,
  write: Record<string, channel.Key>,
  { propertiesKey, channel: current, name, dataType }: CommandChannelSpec,
): Promise<channel.Key> => {
  if (primitive.isNonZero(current)) {
    write[propertiesKey] = current;
    return current;
  }
  const stored = write[propertiesKey];
  if (await channelExists(client, stored)) return stored;
  const cmdCh = await createChannelWithIndex(client, name, dataType);
  write[propertiesKey] = cmdCh.key;
  return cmdCh.key;
};
