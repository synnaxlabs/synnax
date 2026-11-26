// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax as Client } from "@synnaxlabs/client";
import { caseconv, DataType, deep, errors, primitive } from "@synnaxlabs/x";
import { z } from "zod";

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

// Anything that is not a channel key reads as no index.
const storedIndexZ = z.preprocess(
  (v) => (channel.keyZ.safeParse(v).success ? v : 0),
  channel.keyZ,
);

// Pointers and source keys are values, not property names, so they keep their case.
const readChannelPropsZ = z.object({
  index: storedIndexZ,
  channels: caseconv.preserveCase(z.record(z.string(), channel.keyZ)),
});

/** The channels of one read source, as the properties of a device store them. */
export interface ReadChannelProps extends z.infer<typeof readChannelPropsZ> {}

/** The channels that the read sources of a device made, by source key. */
export const readPropertiesZ = caseconv.preserveKeys(
  z.record(z.string(), readChannelPropsZ).default({}),
);

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
  /** Reads for every source that the task plans, and names their new channels. */
  planner: Planner;
  /** What the device stores for the source. Absent when it stores nothing. */
  props?: ReadChannelProps;
  /** The fields of the source. */
  fields: F[];
  /**
   * Prefix of a generated channel name. A new index takes the name of the index field,
   * or `${namePrefix}_time` when the source has none.
   */
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

/** The name of a new index of arrival times, after the prefix of its source. */
export const ARRIVAL_NAME = "time";

/** Holds a name for a channel that deploying creates, which has no key yet. */
const NEW_HOLDER = 0;

/**
 * Reads what the Core holds for a plan of channels, from what planChannels loads, and
 * names new channels as the Core will hold them after deploying. Plan every source of a
 * task with one planner, in deploy order, so that a source sees the names that earlier
 * sources claim.
 */
export interface Planner {
  /** @returns the channel with key, or null when it does not exist or is 0. */
  retrieve: (key: channel.Key) => channel.Channel | null;
  /** Reads the channels that hold each name, or a numbered form of it. */
  readNames: (names: string[]) => void;
  /**
   * @returns the name that a new channel takes for name. A taken name gets the first
   * free number, such as `_2`.
   */
  claim: (name: string) => string;
  /** @returns the numbered name that ch moves to, which frees its own. */
  move: (ch: channel.Channel) => string;
}

/** What planChannels loaded from the Core. */
interface Loaded {
  /** Each loaded key, with null for a key that has no channel. */
  channels: Map<channel.Key, channel.Channel | null>;
  /** The names whose channels are loaded, with those of their numbered forms. */
  names: Set<string>;
  /** The channels that hold each name. Names need not be unique. */
  holders: Map<string, channel.Key[]>;
}

/** What a run of a plan read that was not loaded. */
interface Misses {
  keys: Set<channel.Key>;
  names: Set<string>;
}

const hasMisses = ({ keys, names }: Misses): boolean => keys.size > 0 || names.size > 0;

/**
 * @returns a planner with no claims that reads from loaded and adds each read that
 * loaded lacks to misses. A missed read answers as if no channel holds it.
 */
const createPlanner = (loaded: Loaded, misses: Misses): Planner => {
  const holders = new Map<string, Set<channel.Key>>();
  const holdersOf = (name: string): Set<channel.Key> => {
    let keys = holders.get(name);
    if (keys == null) {
      keys = new Set(loaded.holders.get(name));
      holders.set(name, keys);
    }
    return keys;
  };
  const read = (name: string): void => {
    if (!loaded.names.has(name)) misses.names.add(name);
  };
  const free = (name: string): string => {
    read(name);
    let candidate = name;
    for (let n = 2; holdersOf(candidate).size > 0; n++) candidate = `${name}_${n}`;
    return candidate;
  };
  return {
    retrieve: (key) => {
      if (!primitive.isNonZero(key)) return null;
      const ch = loaded.channels.get(key);
      if (ch === undefined) misses.keys.add(key);
      return ch ?? null;
    },
    readNames: (names) => names.forEach(read),
    claim: (name) => {
      const claimed = free(name);
      holdersOf(claimed).add(NEW_HOLDER);
      return claimed;
    },
    move: (ch) => {
      const moved = free(ch.name);
      holdersOf(ch.name).delete(ch.key);
      holdersOf(moved).add(ch.key);
      return moved;
    },
  };
};

const escapeRegex = (text: string): string =>
  text.replace(/[\\^$.|?*+()[\]{}]/g, "\\$&");

/**
 * Loads what misses holds into loaded, with one read of keys and one of names, and then
 * one read of the indexes of the channels that it loads.
 */
const load = async (client: Client, loaded: Loaded, misses: Misses): Promise<void> => {
  const keys = [...misses.keys];
  const names = [...misses.names];
  // Also matches each numbered form, such as `_2`, that claim and move can give.
  const regex = `^(?:${names.map(escapeRegex).join("|")})(?:_[0-9]+)?$`;
  const [channels, holders] = await Promise.all([
    keys.length === 0 ? [] : client.channels.retrieve(keys),
    names.length === 0 ? [] : client.channels.retrieve({ where: { name: { regex } } }),
  ]);
  for (const key of keys) loaded.channels.set(key, null);
  for (const ch of channels) loaded.channels.set(ch.key, ch);
  const indexes = [
    ...new Set(channels.map(({ index }) => index).filter(primitive.isNonZero)),
  ].filter((key) => !loaded.channels.has(key));
  if (indexes.length > 0) {
    for (const key of indexes) loaded.channels.set(key, null);
    for (const ch of await client.channels.retrieve(indexes))
      loaded.channels.set(ch.key, ch);
  }
  for (const name of names) loaded.names.add(name);
  for (const { name, key } of holders)
    loaded.holders.set(name, [...(loaded.holders.get(name) ?? []), key]);
};

/**
 * Plans with what the Core holds, in two runs of plan. The first run finds what plan
 * reads, one batch loads it with the index of each channel, and the second run gives
 * the result, so it retrieves channels at most 3 times. plan must have no side effects,
 * and must read each key and name that it can need before it branches on what it reads.
 * It can also read the index of a channel that it read.
 *
 * @throws an error when the second run reads something that the first did not, which is
 * a defect of plan.
 */
export const planChannels = async <P>(
  client: Client,
  plan: (planner: Planner) => P,
  signal?: AbortSignal,
): Promise<P> => {
  const loaded: Loaded = { channels: new Map(), names: new Set(), holders: new Map() };
  const first: Misses = { keys: new Set(), names: new Set() };
  const planned = plan(createPlanner(loaded, first));
  if (!hasMisses(first)) return planned;
  await load(client, loaded, first);
  signal?.throwIfAborted();
  const second: Misses = { keys: new Set(), names: new Set() };
  const replanned = plan(createPlanner(loaded, second));
  if (hasMisses(second))
    throw new Error(
      `a channel plan read keys [${[...second.keys].join(", ")}] and names ` +
        `[${[...second.names].join(", ")}] only after its load`,
    );
  return replanned;
};

/**
 * Plans the channel of every enabled field of a read source: the one it has, if it
 * exists, the one the device stores for its pointer, or a new one. The index field or a
 * new indexed channel gets the stored index. When that is gone, it gets the channel of
 * the index field, the index of a live channel of the source, or a new one. A new
 * channel whose name is taken gets a number, such as `_2`, and fields claim their names
 * before a new index. A replaced channel that holds the name of its replacement moves
 * to a numbered name first, so the replacement keeps the plain one. Changes neither the
 * fields nor props.
 */
export const planReadChannels = <F extends ReadChannelField>({
  planner,
  props: stored,
  fields,
  namePrefix,
  indexKey,
  indexed = isFixed,
}: PlanReadChannelsArgs<F>): ReadChannelPlan => {
  const props = stored == null ? { index: 0, channels: {} } : deep.copy(stored);
  let saves = stored != null;
  const enabled = fields.filter(({ disabled }) => !disabled);
  const indexField = enabled.find(({ key }) => key === indexKey);
  const nameOf = (field: F): string =>
    primitive.isNonZero(field.name)
      ? field.name
      : `${namePrefix}${channel.escapeInvalidName(field.pointer)}`;
  const indexName =
    indexField == null ? `${namePrefix}_${ARRIVAL_NAME}` : nameOf(indexField);
  const storedIndex = planner.retrieve(props.index);
  const held = Object.values(props.channels).map((key) => planner.retrieve(key));
  const current = new Map(enabled.map((f) => [f.key, planner.retrieve(f.channel)]));
  const replaced = new Map(
    enabled.map((f) => [f.key, planner.retrieve(f.replaces ?? 0)]),
  );
  planner.readNames([...enabled.map(nameOf), indexName]);
  const channels: Record<string, channel.Key> = {};
  const live: channel.Channel[] = [];
  const bind = (field: F): boolean => {
    const own = current.get(field.key);
    if (own != null && indexed(field)) live.push(own);
    const ch = own ?? planner.retrieve(props.channels[field.pointer]);
    if (ch == null) return false;
    channels[field.key] = ch.key;
    return true;
  };
  const unbound: F[] = [];
  for (const field of enabled)
    if (field !== indexField && !bind(field)) unbound.push(field);
  const needsIndex =
    enabled.some(indexed) && (indexField != null || unbound.some(indexed));
  let createsIndex = false;
  if (needsIndex && storedIndex == null) {
    const own = indexField == null ? null : (current.get(indexField.key) ?? null);
    props.index = recoverIndex(own, [...live, ...held]);
    createsIndex = !primitive.isNonZero(props.index);
    saves = true;
  }
  const takesIndex = createsIndex || primitive.isNonZero(props.index);
  if (indexField != null && !takesIndex && !bind(indexField)) unbound.push(indexField);
  const planned = unbound.map(nameOf);
  const named = unbound.map((field, i): [F, string] => [field, planned[i]]);
  if (createsIndex && indexField != null) named.push([indexField, indexName]);
  const renames: ReadChannelPlan["renames"] = [];
  for (const [field, name] of named) {
    const old = replaced.get(field.key);
    if (old == null || old.name !== name) continue;
    renames.push({ key: old.key, name: planner.move(old) });
  }
  const creates: ReadChannelPlan["creates"] = unbound.map((field, i) => ({
    key: field.key,
    name: planner.claim(planned[i]),
    indexed: indexed(field),
  }));
  const newIndex = createsIndex ? planner.claim(indexName) : undefined;
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

/**
 * @returns own, the channel of the index field, when it is an index. Else the first
 * index among those of channels, or 0.
 */
const recoverIndex = (
  own: channel.Channel | null,
  channels: (channel.Channel | null)[],
): channel.Key => {
  if (own?.isIndex === true) return own.key;
  for (const ch of channels)
    if (ch != null && primitive.isNonZero(ch.index)) return ch.index;
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
  /**
   * The data types of an existing channel that can carry the commands of the target. A
   * current or stored channel of another data type is replaced. When absent, every
   * channel can.
   */
  dataTypes?: string[];
}

/**
 * What deploying does for the command channel of a write target: keep a channel, or
 * create one by name after moving the replaced channel that holds it.
 */
export type CommandChannelPlan =
  | { keeps: channel.Key }
  | { creates: string; moves?: { channel: channel.Channel; name: string } };

const planCommandChannel = (
  planner: Planner,
  write: Record<string, channel.Key>,
  { propertiesKey, channel: current, name, dataTypes }: CommandChannelSpec,
): CommandChannelPlan => {
  if (dataTypes == null && primitive.isNonZero(current)) return { keeps: current };
  const suits = (ch: channel.Channel) =>
    dataTypes == null || dataTypes.includes(ch.dataType.toString());
  const ch = planner.retrieve(current);
  const stored = planner.retrieve(write[propertiesKey]);
  planner.readNames([name]);
  if (ch != null && suits(ch)) return { keeps: current };
  if (stored != null && suits(stored)) return { keeps: stored.key };
  const holder = [ch, stored].find((c) => c?.name === name);
  const moves =
    holder == null ? undefined : { channel: holder, name: planner.move(holder) };
  return { creates: planner.claim(name), moves };
};

/** @returns the name of the channel that plan creates, or undefined if it keeps one. */
export const newCommandChannelName = (
  plan: CommandChannelPlan | undefined,
): string | undefined => (plan != null && "creates" in plan ? plan.creates : undefined);

/**
 * Plans the command channels of the write targets of a task, in deploy order, as
 * configureCommandChannel gives them. Changes nothing.
 *
 * @returns the plan of the first target of each properties key, by that key. A later
 * target of the key with no channel gets the same channel.
 */
export const planCommandChannels = (
  planner: Planner,
  write: Record<string, channel.Key>,
  specs: CommandChannelSpec[],
): Map<string, CommandChannelPlan> => {
  const plans = new Map<string, CommandChannelPlan>();
  for (const spec of specs) {
    // Deploying stores the channel of the first target, so a later one adopts it.
    if (plans.has(spec.propertiesKey)) continue;
    plans.set(spec.propertiesKey, planCommandChannel(planner, write, spec));
  }
  return plans;
};

/**
 * Gives a write target a command channel: the one it has, the one the device stores for
 * its key, or a new one. A replaced channel that holds the name of the new one moves to
 * a numbered name first. Another channel that holds it gives the new one the first free
 * number, such as `_2`.
 *
 * @param write - The write properties of the device. Updated in place.
 * @returns the command channel.
 */
export const configureCommandChannel = async (
  client: Client,
  write: Record<string, channel.Key>,
  spec: CommandChannelSpec,
): Promise<channel.Key> => {
  const plan = await planChannels(client, (planner) =>
    planCommandChannel(planner, write, spec),
  );
  if ("keeps" in plan) {
    write[spec.propertiesKey] = plan.keeps;
    return plan.keeps;
  }
  if (plan.moves != null) {
    const { channel: ch, name } = plan.moves;
    await client.channels.rename(ch.key, name);
    // The index created with the channel moves with it.
    const index = await retrieveChannel(client, ch.index);
    if (index?.name === `${ch.name}_time`)
      await client.channels.rename(index.key, `${name}_time`);
  }
  const cmdCh = await createChannelWithIndex(client, plan.creates, spec.dataType);
  write[spec.propertiesKey] = cmdCh.key;
  return cmdCh.key;
};
