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
 * Creates a Core channel for values of dataType. A fixed-density channel gets its own
 * index named `${name}_time`; a variable-density one is virtual.
 */
export const createChannelWithIndex = async (
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
}

export interface ConfigureReadChannelsArgs<F extends ReadChannelField> {
  client: Client;
  /** The read properties of the device. Updated in place. */
  read: Record<string, ReadChannelProps>;
  /** The key of the source in read. */
  propertiesKey: string;
  /** The fields of the source. Each enabled one gets its channel in place. */
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
 * Gives every enabled field of a read source a channel: the one it has, the one the
 * device stores for its pointer, or a new one, which read then stores. The index field
 * or a new indexed channel gets the stored index. When that is gone, it gets the index
 * of a live channel of the source, or a new one. A source that needs no new channel
 * gets no entry in read.
 */
export const configureReadChannels = async <F extends ReadChannelField>({
  client,
  read,
  propertiesKey,
  fields,
  namePrefix,
  indexKey,
  indexed = isFixed,
}: ConfigureReadChannelsArgs<F>): Promise<void> => {
  const props = read[propertiesKey] ?? { index: 0, channels: {} };
  const enabled = fields.filter(({ disabled }) => !disabled);
  const live: channel.Channel[] = [];
  const bind = async (field: F): Promise<boolean> => {
    const ch = await retrieveChannel(client, field.channel);
    if (ch != null) {
      if (indexed(field)) live.push(ch);
      return true;
    }
    const stored = props.channels[field.pointer];
    if (!(await channelExists(client, stored))) return false;
    field.channel = stored;
    return true;
  };
  const indexField = enabled.find(({ key }) => key === indexKey);
  const unbound: F[] = [];
  for (const field of enabled)
    if (field !== indexField && !(await bind(field))) unbound.push(field);
  const needsIndex =
    enabled.some(indexed) && (indexField != null || unbound.some(indexed));
  if (needsIndex && !(await channelExists(client, props.index))) {
    props.index = await recoverIndex(client, live, props);
    if (!primitive.isNonZero(props.index)) {
      const indexCh = await client.channels.create({
        name: `${namePrefix}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      props.index = indexCh.key;
    }
    read[propertiesKey] = props;
  }
  if (indexField != null && primitive.isNonZero(props.index))
    indexField.channel = props.index;
  else if (indexField != null && !(await bind(indexField))) unbound.push(indexField);
  for (const field of unbound) {
    const name = primitive.isNonZero(field.name)
      ? field.name
      : `${namePrefix}${channel.escapeInvalidName(field.pointer)}`;
    const ch = await client.channels.create({
      name,
      dataType: field.dataType,
      ...(isFixed(field) ? { index: props.index } : { virtual: true }),
    });
    field.channel = ch.key;
    props.channels[field.pointer] = ch.key;
    read[propertiesKey] = props;
  }
};

/**
 * @returns the first index that still exists among those of the live channels, then
 * those of the stored channels, or 0.
 */
const recoverIndex = async (
  client: Client,
  live: channel.Channel[],
  props: ReadChannelProps,
): Promise<channel.Key> => {
  for (const ch of live) if (await channelExists(client, ch.index)) return ch.index;
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
 * @returns the command channel.
 */
export const configureCommandChannel = async (
  client: Client,
  write: Record<string, channel.Key>,
  { propertiesKey, channel: current, name, dataType }: CommandChannelSpec,
): Promise<channel.Key> => {
  if (await channelExists(client, current)) {
    write[propertiesKey] = current;
    return current;
  }
  const stored = write[propertiesKey];
  if (await channelExists(client, stored)) return stored;
  const cmdCh = await createChannelWithIndex(client, name, dataType);
  write[propertiesKey] = cmdCh.key;
  return cmdCh.key;
};
