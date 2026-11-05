// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  array,
  color,
  type CrudeTimeRange,
  deep,
  type destructor,
  primitive,
  type Series,
  TimeRange,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { type channel } from "@/channel";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { createOf, encode, reduceAll } from "@/ranger/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/ranger/actions.gen";
import { alias } from "@/ranger/alias";
import { Client as AliasClient } from "@/ranger/alias/client";
import { kv } from "@/ranger/kv";
import { Client as KVClient } from "@/ranger/kv/client";
import { type Name, type Params } from "@/ranger/payload";
import {
  type Key,
  keyZ,
  type New,
  type Payload,
  payloadZ,
  resourceID,
} from "@/ranger/types.gen";
import { Writer } from "@/ranger/writer";
import { type resource } from "@/resource";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const ACTION_CHANNEL_NAME = "sy_range_action_set";
export const DELETE_CHANNEL_NAME = "sy_range_delete";

const kvDeleteZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

const aliasDeleteZ = z.string().transform((val) => alias.decodeDeleteChange(val));

interface Stores {
  ranges: query.Table<Key, Range>;
  kvPairs: query.Table<string, kv.Pair>;
  aliases: query.Table<string, alias.Alias>;
}

/** Creates the range, range KV, and range alias tables on the given cache. */
const createTables = (
  cache: query.Cache,
  fetch: (keys: Key[]) => Promise<Range[]>,
): Stores => {
  const ranges = cache.createTable<Key, Range>({
    name: "ranges",
    equal: (a, b) => deep.equal(a.payload, b.payload),
    fetch,
    listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
  });

  const kvPairs = cache.createTable<string, kv.Pair>({
    name: "range_kv",
    listen: [
      query.createSetListener(kv.SET_CHANNEL_NAME, kv.pairZ, {
        key: (changed) => kv.createPairKey(changed),
      }),
      query.createDeleteListener(kv.DELETE_CHANNEL_NAME, kvDeleteZ, {
        key: (changed) => kv.createPairKey(changed),
      }),
    ],
  });

  const aliases = cache.createTable<string, alias.Alias>({
    name: "range_aliases",
    listen: [
      query.createSetListener(alias.SET_CHANNEL_NAME, alias.aliasZ, {
        key: (changed) => alias.createKey(changed),
      }),
      query.createDeleteListener(alias.DELETE_CHANNEL_NAME, aliasDeleteZ, {
        key: (changed) => alias.createKey(changed),
      }),
    ],
  });
  return { ranges, kvPairs, aliases };
};

interface RangeConstructionOptions {
  frameClient: framer.Client;
  kv: KVClient;
  aliaser: AliasClient;
  channels: framer.ChannelRetriever;
  labelClient: label.Client;
  rangeClient: Client;
}

/**
 * A named window of time, with the labels, metadata, and channel aliases attached to
 * it. Its `read` and `kv` members scope those operations to its own time range.
 */
export class Range {
  key: string;
  name: string;
  readonly kv: KVClient;
  readonly timeRange: TimeRange;
  readonly color?: color.Color;
  /** The key of the range this range is nested in. */
  readonly parent?: Key;
  /** The keys of the labels attached to the range. */
  readonly labels: label.Key[];
  readonly channels: framer.ChannelRetriever;
  private readonly aliaser: AliasClient;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly rangeClient: Client;

  constructor(
    { name, timeRange = TimeRange.ZERO, key, color: color_, parent, labels }: Payload,
    {
      frameClient,
      kv,
      aliaser,
      channels,
      labelClient,
      rangeClient,
    }: RangeConstructionOptions,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.parent = parent;
    this.labels = labels;
    this.frameClient = frameClient;
    this.color = color_;
    this.kv = kv;
    this.aliaser = aliaser;
    this.channels = channels;
    this.labelClient = labelClient;
    this.rangeClient = rangeClient;
  }

  get resourceID(): resource.ID {
    return resourceID(this.key);
  }

  get payload(): Payload {
    const r: Payload = {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
      labels: this.labels,
    };
    if (this.parent != null) r.parent = this.parent;
    return r;
  }

  async setAlias(channel: channel.Key | Name, alias: string): Promise<void> {
    const ch = await this.channels(channel);
    if (ch.length === 0) throw new QueryError(`Channel ${channel} does not exist`);
    await this.rangeClient.setAlias(this.key, ch[0].key, alias);
  }

  async deleteAlias(...channels: channel.Key[]): Promise<void> {
    await this.rangeClient.deleteAlias(this.key, channels);
  }

  async listAliases(): Promise<Record<channel.Key, string>> {
    return await this.aliaser.list();
  }

  async resolveAlias(alias: string): Promise<channel.Key> {
    return await this.aliaser.resolve(alias);
  }

  async retrieveParent(): Promise<Range | null> {
    if (this.parent == null) return null;
    return await this.rangeClient.retrieve(this.parent);
  }

  async retrieveChildren(): Promise<Range[]> {
    return await this.rangeClient.retrieve({ parents: [this.key] });
  }

  async read(channel: Key | Name): Promise<Series>;
  async read(channels: Params): Promise<framer.Frame>;
  async read(channels: Params): Promise<Series | framer.Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }

  async retrieveLabels(): Promise<label.Label[]> {
    if (this.labels.length === 0) return [];
    return await this.labelClient.retrieve({ keys: this.labels });
  }

  async addLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.label(resourceID(this.key), labels);
  }

  async removeLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.remove(resourceID(this.key), labels);
  }

  static sort(a: Range, b: Range): number {
    return TimeRange.sort(a.timeRange, b.timeRange);
  }
}

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  overlapsWith: TimeRange.z.optional(),
  hasLabels: label.keyZ.array().optional(),
  parents: keyZ.array().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});

/** Everything a range retrieval can filter on. */
export type RetrieveRequest = z.infer<typeof retrieveRequestZ>;

const retrieveParamsZ = retrieveRequestZ
  .or(keyZ.array().transform((keys) => ({ keys })))
  .or(keyZ.transform((key) => ({ keys: [key] })))
  .or(z.string().transform((name) => ({ names: [name] })))
  .or(
    z
      .string()
      .array()
      .transform((names) => ({ names })),
  )
  .or(TimeRange.z.transform((timeRange) => ({ overlapsWith: timeRange })));

/** Params for a range retrieval. A bare key, name, or time range is shorthand. */
export type RetrieveParams = z.input<typeof retrieveParamsZ>;

/** Canonicalizes every retrieve shape addressing more than one range. */
const retrieveMultiParamsZ = retrieveRequestZ
  .or(keyZ.array().transform((keys) => ({ keys })))
  .or(
    z
      .string()
      .array()
      .transform((names) => ({ names })),
  )
  .or(TimeRange.z.transform((timeRange) => ({ overlapsWith: timeRange })));

const retrieveResZ = z.object({ ranges: payloadZ.array().default(() => []) });

/**
 * Request addressing a range's children. With an `offset`, it addresses a page in the
 * order the Core returns them. A `limit` alone addresses some `limit` children, not
 * necessarily the first.
 */
export type ChildrenRequest = {
  key: Key;
  limit?: number;
  offset?: number;
};

/** Params addressing a range's children. A bare key addresses all of them. */
export type ChildrenParams = Key | ChildrenRequest;

const CHILDREN_SERVER_FIELDS = ["limit", "offset"] as const;

const normalizeChildren = (params: ChildrenParams): ChildrenRequest =>
  typeof params === "string" ? { key: params } : params;

/** Exact client-side evaluation of a request against a cached range. */
const requestMatches = (r: Range, req: RetrieveRequest): boolean => {
  if (primitive.isNonZero(req.keys) && !req.keys.includes(r.key)) return false;
  if (primitive.isNonZero(req.names) && !req.names.includes(r.name)) return false;
  if (
    req.overlapsWith != null &&
    !new TimeRange(req.overlapsWith).overlapsWith(r.timeRange)
  )
    return false;
  if (
    primitive.isNonZero(req.hasLabels) &&
    !req.hasLabels.some((l) => r.labels.includes(l))
  )
    return false;
  if (
    primitive.isNonZero(req.parents) &&
    (r.parent == null || !req.parents.includes(r.parent))
  )
    return false;
  return true;
};

const isParentChange = (rel: ontology.Relationship, id: resource.ID): boolean =>
  ontology.matchRelationship(rel, {
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: id,
  });

/** Config for {@link Client}. */
export interface ClientConfig {
  framer: framer.Client;
  unary: UnaryClient;
  channels: framer.ChannelRetriever;
  labels: label.Client;
  ontology: ontology.Client;
  cache: query.Cache;
}

/**
 * Creates, reads, and deletes ranges on a Core. Reach it through `client.ranges`. Reads
 * return sugared {@link Range}s, served from a cache a change stream keeps current.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Range,
  Range,
  Key | Name
> {
  readonly type: string = "range";
  /** The range alias table; injected into sibling clients at wiring. */
  readonly aliases: query.Table<string, alias.Alias>;
  /** Cached queries for the children of a range, keyed by the parent's key. */
  readonly children: query.Retrieves<ChildrenParams, Range[]>;
  /**
   * Cached queries for the closest range parent of a resource, keyed by the
   * child's resource ID.
   */
  readonly parent: query.Retrieves<resource.ID, Range | null>;
  /** Cached queries for a range's KV metadata pairs, keyed by the range's key. */
  readonly kv: query.Retrieves<Key, kv.Pair[]>;
  private readonly cfg: ClientConfig;
  private readonly writer: Writer;
  private readonly store: query.Table<Key, Range>;
  private readonly dispatcher: actions.Controller<Key, Range, Action>;
  private readonly kvPairs: query.Table<string, kv.Pair>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const { ranges, kvPairs, aliases } = createTables(
      cache,
      async (keys) => await this.execRetrieve({ keys, ignoreNotFoundError: true }),
    );
    // Every write to a range reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Range, Action>({
      store: ranges,
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      send: actions.sender(cfg.unary, "/range/dispatch", dispatchReqZ, dispatchResZ),
      reduce: (range, acts) => {
        const { payload } = range;
        const { next, inverse, targets } = reduceAll(payload, acts);
        return {
          next: next === payload ? range : this.sugarOne(next),
          inverse,
          targets,
        };
      },
      encode: (range, action) => encode(range.payload, action),
      createOf: (action) => {
        const created = createOf(action);
        return created == null ? undefined : this.sugarOne(created);
      },
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    const single = cache.queries<Key | Name, Range, Key, Range>({
      name: "range",
      table: ranges,
      fetch: async (query) => [(await this.fetchSingle(query)).key],
      compose: ([record]) => record,
      keyOf: (query) => (z.validate(keyZ, query) ? query : null),
      matches: (r, query) => r.key === query || r.name === query,
      single: true,
    });
    super(cache, {
      name: "range",
      table: ranges,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (query) => await this.execRetrieve(query),
        matches: requestMatches,
      },
      single: { schema: z.string(), space: single },
    });
    this.cfg = cfg;
    this.writer = new Writer(cfg.unary);
    this.store = ranges;
    this.dispatcher = dispatcher;
    this.kvPairs = kvPairs;
    this.aliases = aliases;
    const children = cache.queries<ChildrenRequest, Range[], Key, Range>({
      name: "child ranges",
      table: ranges,
      fetch: async ({ key, limit, offset }) => {
        const since = ranges.stamp();
        const found = await this.execRetrieve({ parents: [key], limit, offset });
        ranges.ingest(found, { since });
        return found.map((r) => r.key);
      },
      compose: (records) => records,
      matches: (r, query) => r.parent === query.key,
      serverFields: CHILDREN_SERVER_FIELDS,
    });
    this.children = {
      retrieve: async (params, options) =>
        await children.retrieve(normalizeChildren(params), options),
      onChange: (params, handler) =>
        children.onChange(normalizeChildren(params), handler),
      getCached: (params) => children.getCached(normalizeChildren(params)),
    };
    this.parent = cache.queries<resource.ID, Range | null, Key, Range>({
      name: "parent range",
      table: ranges,
      fetch: async (query) => {
        const range = await this.fetchParent(query);
        return range == null ? [] : [range.key];
      },
      compose: (records) => records[0] ?? null,
      matches: (r, query) => this.parentKeyOf(query) === r.key,
      watch: [
        // A change to the child range can move it to another parent.
        query.watch<resource.ID, Key, Key, Range>(ranges, (event, query) =>
          query.type === "range" && event.key === query.key ? "refetch" : null,
        ),
        query.watch<resource.ID, Key, string, ontology.Relationship>(
          cfg.ontology.cache.relationships,
          (event, query) => {
            if (query.type === "range") return null;
            const rel =
              event.variant === "set"
                ? event.value
                : zod.parse(ontology.relationshipZ, event.key, {
                    label: "ontology relationship",
                  });
            return isParentChange(rel, query) ? "refetch" : null;
          },
        ),
      ],
    });
    this.kv = cache.queries<Key, kv.Pair[], string, kv.Pair>({
      name: "range metadata",
      table: this.kvPairs,
      fetch: async (query) =>
        (await this.fetchKV(query)).map((p) => kv.createPairKey(p)),
      compose: (records) => records,
      matches: (pair, query) => pair.range === query,
    });
  }

  /**
   * Creates the ranges. Creating a range that exists keeps the labels it holds and
   * ignores the carried ones, and keeps its parent unless one is carried. Change labels
   * with the label client.
   */
  async create(range: New): Promise<Range>;
  async create(ranges: New[]): Promise<Range[]>;
  async create(ranges: New | New[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const news = array.toArray(ranges);
    const keys = news.flatMap(({ key }) => (key == null ? [] : [key]));
    const res = this.sugarMany(
      await this.dispatcher.create(
        keys,
        async (dispatchKey) => await this.writer.create(news, dispatchKey),
      ),
    );
    this.store.set(res);
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      this.store.set(key, (p) =>
        p == null ? undefined : this.sugarOne({ ...p.payload, name }),
      ),
      this.cfg.ontology.cache.renameResource(resourceID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.writer.rename(key, name),
    });
    rename();
  }

  /**
   * Applies actions to the cached range and sends them to the Core. Paths and values
   * take the camelCase form of the cached range. Returns false without side effects
   * when the range is not cached.
   * @throws {Error} if the range cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached range drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  /** Deletes the ranges and every range nested in them. */
  async delete(key: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keys = array.toArray(key);
    const drop = () => [
      this.cfg.ontology.cache.deleteRelationships(resourceID(keys)),
      this.store.delete(keys),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.writer.delete(keys),
    });
    drop();
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(
    params: Key[] | Name[] | CrudeTimeRange | RetrieveRequest,
  ): Promise<Range[]>;
  async retrieve(params: RetrieveParams): Promise<Range | Range[]> {
    // The branches narrow params onto different base overloads.
    if (typeof params === "string") return await super.retrieve(params);
    return await super.retrieve(params);
  }

  private async execRetrieve(params: RetrieveParams): Promise<Range[]> {
    const { ranges } = await this.cfg.unary.send(
      "/range/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return this.sugarMany(ranges);
  }

  /** Subscribes to every range set delivered to the cache. */
  onSet(handler: (range: Range) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every range delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private createAliasClient(key: Key): AliasClient {
    return new AliasClient(key, this.cfg.unary);
  }

  private createKVClient(key: Key): KVClient {
    return new KVClient(key, this.cfg.unary, this.kvPairs);
  }

  private async fetchSingle(query: Key | Name): Promise<Range> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    if (z.validate(keyZ, query)) {
      const ranges = await this.store.retrieve([query]);
      checkForMultipleOrNoResults("Range", query, ranges, true);
      return ranges[0];
    }
    const since = this.store.stamp();
    const ranges = await this.execRetrieve({ names: [query] });
    checkForMultipleOrNoResults("Range", query, ranges, true);
    this.store.ingest(ranges, { since });
    return ranges[0];
  }

  /** Returns the key of the parent range of the resource, if the cache holds it. */
  private parentKeyOf(id: resource.ID): Key | undefined {
    if (id.type === "range") return this.store.get(id.key)?.parent;
    const parent = this.cfg.ontology.cache.parentID(id);
    return parent?.type === "range" ? parent.key : undefined;
  }

  private async fetchParent(id: resource.ID): Promise<Range | null> {
    let parent: Key | undefined;
    if (id.type === "range") {
      const [child] = await this.store.retrieve([id.key]);
      parent = child?.parent;
    } else {
      const res = await this.cfg.ontology.parents.retrieve({ ids: id });
      parent = res.find(({ id: { type } }) => type === "range")?.id.key;
    }
    if (parent == null) return null;
    const [range] = await this.store.retrieve([parent]);
    return range ?? null;
  }

  private async fetchKV(query: Key): Promise<kv.Pair[]> {
    const pairs = await this.createKVClient(query).list();
    const result: kv.Pair[] = Object.entries(pairs).map(([key, value]) => ({
      range: query,
      key,
      value,
    }));
    result.forEach((p) => this.kvPairs.set(kv.createPairKey(p), p));
    return result;
  }

  getKV(range: Key): KVClient {
    return this.createKVClient(range);
  }

  /** Returns the parent of the given range, or null when it has none. */
  async retrieveParent(range: Key): Promise<Range | null> {
    return await (await this.retrieve(range)).retrieveParent();
  }

  sugarOntologyResource(resource: ontology.Resource): Range {
    return this.sugarOne(convertOntologyResourceToPayload(resource));
  }

  async retrieveAlias(range: Key, channel: channel.Key): Promise<string> {
    const aliaser = this.createAliasClient(range);
    return await aliaser.retrieve(channel);
  }

  async retrieveAliases(
    range: Key,
    channels: channel.Key[],
  ): Promise<Record<channel.Key, string>> {
    return await this.createAliasClient(range).retrieve(channels);
  }

  async listAliases(range: Key): Promise<Record<channel.Key, string>> {
    return await this.createAliasClient(range).list();
  }

  async setAlias(
    range: Key,
    channel: channel.Key,
    aliasName: string,
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const entry: alias.Alias = { range, channel, alias: aliasName };
    const set = () => [this.aliases.set(alias.createKey(entry), entry)];
    await query.optimistic({
      rollbacks: set(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.createAliasClient(range).set({ [channel]: aliasName }),
    });
    set();
  }

  async deleteAlias(
    range: Key,
    channels: channel.Key | channel.Key[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const channelsArr = array.toArray(channels);
    const drop = () => [
      this.aliases.delete(
        channelsArr.map((channel) => alias.createKey({ range, channel })),
      ),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.createAliasClient(range).delete(channelsArr),
    });
    drop();
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.cfg.framer,
      kv: this.createKVClient(payload.key),
      aliaser: this.createAliasClient(payload.key),
      channels: this.cfg.channels,
      labelClient: this.cfg.labels,
      rangeClient: this,
    });
  }

  sugarMany(payloads: Payload[]): Range[] {
    return payloads.map((payload) => this.sugarOne(payload));
  }

  resourceToRange(resource: ontology.Resource): Range {
    return this.sugarOne(convertOntologyResourceToPayload(resource));
  }
}

/** @returns the resource ID of a range's alias set. */
export const aliasResourceID = (key: Key): resource.ID => ({
  type: "range-alias",
  key,
});

/**
 * Rebuilds a range payload from its ontology resource, for a search result that arrives
 * through the ontology rather than the range endpoint.
 */
export const convertOntologyResourceToPayload = ({
  data,
  id: { key },
  name,
}: ontology.Resource): Payload => {
  const timeRange = zod.parse(TimeRange.z, data?.timeRange, {
    label: "range time range",
  });
  const c = color.colorZ.safeParse(data?.color);
  return {
    key,
    name,
    timeRange,
    color: c.success ? c.data : undefined,
    labels: [],
    parent: undefined,
  };
};
