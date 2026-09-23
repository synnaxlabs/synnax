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
export const retrieveChannel = async (
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
export const channelExists = async (
  client: Client,
  key: channel.Key,
): Promise<boolean> => (await retrieveChannel(client, key)) != null;

/**
 * Creates a channel for values of dataType. A fixed-density channel gets its own
 * index named `${name}_time`; a variable-density one is virtual.
 */
export const createChannel = async (
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
}

export interface ConfigureReadChannelsArgs<F extends ReadChannelField> {
  client: Client;
  /** The stored channels of the source. Updated in place. */
  props: ReadChannelProps;
  /** The fields of the source. Each gets its channel in place. */
  fields: F[];
  /** Prefix of a generated channel name. The index is named `${namePrefix}_time`. */
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
 * Gives every field of a read source a channel: the one it has, the one the device
 * stores for its pointer, or a new one. An index that no longer exists is recovered
 * from a stored data channel, or created.
 *
 * @returns true when it changed props.
 */
export const configureReadChannels = async <F extends ReadChannelField>({
  client,
  props,
  fields,
  namePrefix,
  indexKey,
  indexed = isFixed,
}: ConfigureReadChannelsArgs<F>): Promise<boolean> => {
  let modified = false;
  if (fields.some(indexed) && !(await channelExists(client, props.index))) {
    modified = true;
    props.index = await recoverIndex(client, props);
    if (!primitive.isNonZero(props.index)) {
      const indexCh = await client.channels.create({
        name: `${namePrefix}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      props.index = indexCh.key;
    }
  }
  for (const field of fields) {
    if (field.key === indexKey && primitive.isNonZero(props.index)) {
      field.channel = props.index;
      continue;
    }
    if (await channelExists(client, field.channel)) continue;
    const stored = props.channels[field.pointer];
    if (await channelExists(client, stored)) {
      field.channel = stored;
      continue;
    }
    const name = primitive.isNonZero(field.name)
      ? field.name
      : `${namePrefix}${channel.escapeInvalidName(field.pointer)}`;
    const ch = await client.channels.create({
      name,
      dataType: field.dataType,
      ...(isFixed(field) ? { index: props.index } : { virtual: true }),
    });
    modified = true;
    field.channel = ch.key;
    props.channels[field.pointer] = ch.key;
  }
  return modified;
};

/** @returns the index of the first stored data channel that still exists, or 0. */
const recoverIndex = async (
  client: Client,
  props: ReadChannelProps,
): Promise<channel.Key> => {
  for (const stored of Object.values(props.channels)) {
    const ch = await retrieveChannel(client, stored);
    if (ch == null || !(await channelExists(client, ch.index))) continue;
    return ch.index;
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
 * @returns the command channel, and true when it changed write.
 */
export const configureCommandChannel = async (
  client: Client,
  write: Record<string, channel.Key>,
  { propertiesKey, channel: current, name, dataType }: CommandChannelSpec,
): Promise<[channel.Key, boolean]> => {
  if (await channelExists(client, current)) {
    const changed = write[propertiesKey] !== current;
    write[propertiesKey] = current;
    return [current, changed];
  }
  const stored = write[propertiesKey];
  if (await channelExists(client, stored)) return [stored, false];
  const cmdCh = await createChannel(client, name, dataType);
  write[propertiesKey] = cmdCh.key;
  return [cmdCh.key, true];
};
