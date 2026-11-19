// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax as Client } from "@synnaxlabs/client";
import { DataType, deep, errors, primitive } from "@synnaxlabs/x";

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

export interface PlanReadChannelsArgs<F extends ReadChannelField> {
  client: Client;
  /** What the device stores for the source. Absent when it stores nothing. */
  props?: ReadChannelProps;
  /** The fields of the source. */
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

/** The channels that deploying gives a read source, found with reads alone. */
export interface ReadChannelPlan {
  /**
   * What the device stores for the source before deploying creates a channel. Null
   * when the device stores nothing for it until then.
   */
  props: ReadChannelProps | null;
  /** The name of the index that deploying creates, if it creates one. */
  newIndex?: string;
  /** The key of the field whose channel is the index of the source. */
  indexField?: string;
  /** The existing channel of each other enabled field, by key. */
  channels: Record<string, channel.Key>;
  /**
   * The channels that deploying creates for fields, in order. An indexed one stores its
   * samples on the index of the source.
   */
  creates: { key: string; name: string; indexed: boolean }[];
  /** Replaced channels that deploying moves to a numbered name before it creates. */
  renames: { key: channel.Key; name: string }[];
}

const isFixed = (field: ReadChannelField): boolean =>
  !new DataType(field.dataType).isVariable;

/** Holds a name for a channel that deploying creates, which has no key yet. */
const NEW_HOLDER = 0;

/**
 * Names channels as the Core will hold them after deploying. A taken name gets the
 * first free number, such as `_2`. Names need not be unique, so it tracks each holder.
 *
 * @param names - The names to look up in one batch.
 */
const createNamer = async (client: Client, names: string[]) => {
  const holders = new Map<string, Set<channel.Key>>(
    names.map((name) => [name, new Set()]),
  );
  if (names.length > 0)
    for (const { name, key } of await client.channels.retrieve(names))
      holders.get(name)?.add(key);
  const holdersOf = async (name: string): Promise<Set<channel.Key>> => {
    let keys = holders.get(name);
    if (keys == null) {
      const found = await client.channels.retrieve([name]);
      keys = new Set(found.map(({ key }) => key));
      holders.set(name, keys);
    }
    return keys;
  };
  const free = async (name: string): Promise<string> => {
    let candidate = name;
    for (let n = 2; (await holdersOf(candidate)).size > 0; n++)
      candidate = `${name}_${n}`;
    return candidate;
  };
  return {
    /** @returns the name that a new channel takes for name. */
    claim: async (name: string): Promise<string> => {
      const claimed = await free(name);
      (await holdersOf(claimed)).add(NEW_HOLDER);
      return claimed;
    },
    /** @returns the numbered name that ch moves to, which frees its own. */
    move: async (ch: channel.Channel): Promise<string> => {
      const moved = await free(ch.name);
      (await holdersOf(ch.name)).delete(ch.key);
      (await holdersOf(moved)).add(ch.key);
      return moved;
    },
  };
};

/**
 * Plans the channel of every enabled field of a read source: the one it has, the one
 * the device stores for its pointer, or a new one. The index field or a new indexed
 * channel gets the stored index. When that is gone, it gets the index of a live channel
 * of the source, or a new one. A new channel whose name is taken gets a number, such as
 * `_2`, and fields claim their names before a new index. A replaced channel that holds
 * the name of its replacement moves to a numbered name first, so the replacement keeps
 * the plain one. Reads from the Core, and changes neither the fields nor props.
 */
export const planReadChannels = async <F extends ReadChannelField>({
  client,
  props: stored,
  fields,
  namePrefix,
  indexKey,
  indexed = isFixed,
}: PlanReadChannelsArgs<F>): Promise<ReadChannelPlan> => {
  const props = stored == null ? { index: 0, channels: {} } : deep.copy(stored);
  let saves = stored != null;
  const channels: Record<string, channel.Key> = {};
  const enabled = fields.filter(({ disabled }) => !disabled);
  const live: channel.Channel[] = [];
  const bind = async (field: F): Promise<boolean> => {
    if (primitive.isNonZero(field.channel)) {
      if (indexed(field)) live.push(await client.channels.retrieve(field.channel));
      channels[field.key] = field.channel;
      return true;
    }
    const ch = props.channels[field.pointer];
    if (!(await channelExists(client, ch))) return false;
    channels[field.key] = ch;
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
    saves = true;
  }
  const takesIndex = createsIndex || primitive.isNonZero(props.index);
  if (indexField != null && !takesIndex && !(await bind(indexField)))
    unbound.push(indexField);
  const planned = unbound.map((field) =>
    primitive.isNonZero(field.name)
      ? field.name
      : `${namePrefix}${channel.escapeInvalidName(field.pointer)}`,
  );
  const indexName = `${namePrefix}_time`;
  const namer = await createNamer(client, [
    ...planned,
    ...(createsIndex ? [indexName] : []),
  ]);
  const renames: ReadChannelPlan["renames"] = [];
  for (const [i, field] of unbound.entries()) {
    const old = await retrieveChannel(client, field.replaces ?? 0);
    if (old == null || old.name !== planned[i]) continue;
    renames.push({ key: old.key, name: await namer.move(old) });
  }
  const creates: ReadChannelPlan["creates"] = [];
  for (const [i, field] of unbound.entries())
    creates.push({
      key: field.key,
      name: await namer.claim(planned[i]),
      indexed: indexed(field),
    });
  const newIndex = createsIndex ? await namer.claim(indexName) : undefined;
  return {
    props: saves ? props : null,
    newIndex,
    indexField: indexField != null && takesIndex ? indexField.key : undefined,
    channels,
    creates,
    renames,
  };
};

/**
 * Renames and creates the channels of plan, gives each enabled field its channel, and
 * stores the props of plan in stored. Stores each new channel as it creates it, so a
 * failure keeps the channels created before it.
 *
 * @param fields - The fields that plan was made for. Each enabled one gets its channel
 * in place.
 */
export const applyReadChannels = async (
  client: Client,
  plan: ReadChannelPlan,
  fields: ReadChannelField[],
  stored?: StoredChannels,
): Promise<void> => {
  for (const { key, name } of plan.renames) await client.channels.rename(key, name);
  const props = deep.copy(plan.props ?? { index: 0, channels: {} });
  const save = () => {
    if (stored != null) stored.read[stored.key] = props;
  };
  if (plan.props != null) save();
  if (plan.newIndex != null) {
    const index = await client.channels.create({
      name: plan.newIndex,
      dataType: "timestamp",
      isIndex: true,
    });
    props.index = index.key;
  }
  const created = new Map<string, channel.Key>();
  for (const { key, name, indexed } of plan.creates) {
    const field = fields.find((f) => f.key === key);
    if (field == null) throw new Error(`no field has the key ${key}`);
    const ch = await client.channels.create({
      name,
      dataType: field.dataType,
      ...(indexed ? { index: props.index } : { virtual: true }),
    });
    created.set(key, ch.key);
    props.channels[field.pointer] = ch.key;
    save();
  }
  for (const field of fields) {
    const ch =
      field.key === plan.indexField
        ? props.index
        : (plan.channels[field.key] ?? created.get(field.key));
    if (ch != null) field.channel = ch;
  }
};

/**
 * @returns the name of the channel that deploying creates for the field with key, or
 * undefined when the field keeps a channel.
 */
export const newChannelName = (
  plan: ReadChannelPlan,
  key: string,
): string | undefined =>
  key === plan.indexField
    ? plan.newIndex
    : plan.creates.find((create) => create.key === key)?.name;

export interface ConfigureReadChannelsArgs<F extends ReadChannelField> extends Omit<
  PlanReadChannelsArgs<F>,
  "props"
> {
  /** Where the device stores the channels of the source. Absent when it stores none. */
  stored?: StoredChannels;
}

/**
 * Plans the channels of a read source, as planReadChannels does, then applies the plan.
 * Each enabled field gets its channel in place, and stored gets the new channels.
 */
export const configureReadChannels = async <F extends ReadChannelField>({
  stored,
  ...args
}: ConfigureReadChannelsArgs<F>): Promise<void> => {
  const plan = await planReadChannels({ ...args, props: stored?.read[stored.key] });
  await applyReadChannels(args.client, plan, args.fields, stored);
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
