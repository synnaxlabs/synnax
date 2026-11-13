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
  type color,
  type CrudeTimeRange,
  deep,
  type destructor,
  filter,
  type Series,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { type framer } from "@/framer";
import { type label } from "@/label";
import { query } from "@/query";
import { createOf, encode, parentOf, reduceAll } from "@/ranger/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/ranger/actions.gen";
import { kv } from "@/ranger/kv";
import { Client as KVClient } from "@/ranger/kv/client";
import { type Name, type Params } from "@/ranger/payload";
import {
  type Key,
  keyZ,
  type New,
  type Payload,
  resourceID,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/ranger/types.gen";
import { LINES } from "@/ranger/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/ranger/where.gen";
import { Writer } from "@/ranger/writer";
import { type resource } from "@/resource";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const ACTION_CHANNEL_NAME = "sy_range_action_set";
export const DELETE_CHANNEL_NAME = "sy_range_delete";

const kvDeleteZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

interface Stores {
  ranges: query.Table<Key, Range>;
  kvPairs: query.Table<string, kv.Pair>;
}

/** Creates the range and range KV tables on the given cache. */
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
  return { ranges, kvPairs };
};

interface RangeConstructionOptions {
  frameClient: framer.Client;
  kv: KVClient;
  labelClient: label.Client;
  rangeClient: Client;
}

/**
 * A named window of time, with the labels and metadata attached to it. Its `read` and
 * `kv` members scope those operations to its own time range.
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
  /**
   * Whether a range is nested in the range. It can stay true after the last nested
   * range leaves, until the range is retrieved again.
   */
  readonly hasChildren: boolean;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly rangeClient: Client;

  constructor(
    {
      name,
      timeRange = TimeRange.ZERO,
      key,
      color: color_,
      parent,
      labels,
      hasChildren,
    }: Payload,
    { frameClient, kv, labelClient, rangeClient }: RangeConstructionOptions,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.parent = parent;
    this.labels = labels;
    this.hasChildren = hasChildren;
    this.frameClient = frameClient;
    this.color = color_;
    this.kv = kv;
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
      hasChildren: this.hasChildren,
    };
    if (this.parent != null) r.parent = this.parent;
    return r;
  }

  async retrieveParent(): Promise<Range | null> {
    const { parent } = await this.rangeClient.retrieve(this.key);
    if (parent == null) return null;
    return await this.rangeClient.retrieve(parent);
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
    const { labels } = await this.rangeClient.retrieve(this.key);
    if (labels.length === 0) return [];
    return await this.labelClient.retrieve({ key: labels });
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

export type RetrieveMultipleParams = z.infer<typeof retrieveRequestZ>;

/**
 * Canonicalizes every retrieve shape addressing more than one range. The time range arm
 * comes before the request, which would read a time range as an empty request.
 */
const retrieveMultiParamsZ = z.union([
  TimeRange.z.transform((overlaps): RetrieveMultipleParams => ({
    where: { overlaps },
  })),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);

type OneParams = query.OneInput<typeof retrieveMultiParamsZ>;

const isOneParams = (params: object): params is OneParams => query.addressesOne(params);

/** Config for {@link Client}. */
export interface ClientConfig {
  framer: framer.Client;
  unary: UnaryClient;
  labels: label.Client;
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
  { key: Key }
> {
  readonly type: string = "range";
  /** Cached queries for the parent of a range, keyed by the child's key. */
  readonly parent: query.Retrieves<Key, Range | null>;
  /** Cached queries for a range's KV metadata pairs, keyed by the range's key. */
  readonly kv: query.Retrieves<Key, kv.Pair[]>;
  private readonly cfg: ClientConfig;
  private readonly writer: Writer;
  private readonly store: query.Table<Key, Range>;
  private readonly dispatcher: actions.Controller<Key, Range, Action>;
  private readonly kvPairs: query.Table<string, kv.Pair>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const { ranges, kvPairs } = createTables(
      cache,
      async (keys) => await this.execRetrieve(query.keysRequest(keys)),
    );
    // Every write to a range reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Range, Action>({
      store: ranges,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
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
    const listener = dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ);
    // The Core derives hasChildren only on read, so a range that a create or a move
    // places under a parent marks the parent here. The parent keeps the mark after its
    // last child leaves.
    cache.listen({
      ...listener,
      onChange: (frames) => {
        listener.onChange(frames);
        frames.forEach(({ actions }) =>
          actions.forEach((action) => {
            const parent = parentOf(action);
            if (parent != null) this.markParent(parent);
          }),
        );
      },
    });
    const single = cache.queries<Key, Range, Key, Range>({
      name: "range",
      table: ranges,
      fetch: async (key) => [(await this.fetchSingle(key)).key],
      compose: ([record]) => record,
      keyOf: (key) => key,
      single: true,
    });
    super(cache, {
      name: "range",
      table: ranges,
      shortcut: shortcutZ,
      request: {
        list: {
          cursor: (r, order) => cursor(r.payload, order),
          compare: compareCursors,
        },
        schema: retrieveMultiParamsZ,
        fetch: async (query) => await this.execRetrieve(query),
        matches: (r, query) => matches(LINES, r.payload, query.where),
        serverComputed: (query) => serverComputed(LINES, query.where),
      },
      single: {
        schema: z.union([
          keyZ,
          z.strictObject({ key: keyZ }).transform(({ key }) => key),
        ]),
        space: single,
      },
    });
    this.cfg = cfg;
    this.writer = new Writer(cfg.unary);
    this.store = ranges;
    this.dispatcher = dispatcher;
    this.kvPairs = kvPairs;
    this.parent = cache.queries<Key, Range | null, Key, Range>({
      name: "parent range",
      table: ranges,
      fetch: async (query) => {
        const range = await this.fetchParent(query);
        return range == null ? [] : [range.key];
      },
      compose: (records) => records[0] ?? null,
      matches: (r, query) => this.store.get(query)?.parent === r.key,
      triggers: [
        // A change to the child range can move it to another parent.
        query.trigger<Key, Key, Key, Range>(ranges, (event, query) =>
          event.key === query ? "refetch" : null,
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
   * Creates the ranges. Change labels with the label client.
   * @throws {ValidationError} if a range with a carried key exists.
   */
  async create(range: New): Promise<Range>;
  async create(ranges: New[]): Promise<Range[]>;
  async create(ranges: New | New[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const news = array.toArray(ranges);
    const keys = news.flatMap(({ key }) => (key == null ? [] : [key]));
    // A label change or a child streamed during the request is newer than the reply.
    const res = this.sugarMany(
      await this.dispatcher.create(
        keys,
        async (dispatchKey) => await this.writer.create(news, dispatchKey),
      ),
    ).map((r) => {
      const cached = this.store.get(r.key);
      return cached == null
        ? r
        : this.sugarOne({
            ...r.payload,
            labels: cached.labels,
            hasChildren: cached.hasChildren,
          });
    });
    this.store.set(res);
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Changes the labels of the cached range under the key with update, and returns the
   * rollback. It skips a range the cache does not hold.
   */
  relabel(
    key: Key,
    update: (labels: label.Key[]) => label.Key[],
  ): destructor.Destructor {
    return this.store.set(key, (p) =>
      p == null ? undefined : this.sugarOne({ ...p.payload, labels: update(p.labels) }),
    );
  }

  private markParent(key: Key): void {
    this.store.set(key, (p) =>
      p == null || p.hasChildren
        ? p
        : this.sugarOne({ ...p.payload, hasChildren: true }),
    );
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
    const drop = () => [this.store.delete(keys)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.writer.delete(keys),
    });
    drop();
  }

  async retrieve(params: Key | Name | OneParams): Promise<Range>;
  async retrieve(
    params: Name[] | CrudeTimeRange | query.RequestInput<typeof retrieveMultiParamsZ>,
  ): Promise<Range[]>;
  async retrieve(
    params:
      | Key
      | Name
      | OneParams
      | Name[]
      | CrudeTimeRange
      | query.RequestInput<typeof retrieveMultiParamsZ>,
  ): Promise<Range | Range[]> {
    // The branches narrow params onto different base overloads.
    if (typeof params === "string" || isOneParams(params))
      return await super.retrieve(params);
    return await super.retrieve(params);
  }

  private async execRetrieve(req: RetrieveMultipleParams): Promise<Range[]> {
    const { ranges } = await this.cfg.unary.send(
      "/range/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
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

  private createKVClient(key: Key): KVClient {
    return new KVClient(key, this.cfg.unary, this.kvPairs);
  }

  private async fetchSingle(key: Key): Promise<Range> {
    const cached = this.store.get(key);
    if (cached != null) return cached;
    const ranges = await this.store.retrieve([key]);
    checkForMultipleOrNoResults("range", { where: { key: { eq: key } } }, ranges, true);
    return ranges[0];
  }

  private async fetchParent(key: Key): Promise<Range | null> {
    const [child] = await this.store.retrieve([key]);
    const parent = child?.parent;
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

  /**
   * Returns the parent of the given range, or null when it has none. Throws a
   * NotFoundError when the range does not exist.
   */
  async retrieveParent(range: Key): Promise<Range | null> {
    await this.retrieve(range);
    return await this.parent.retrieve(range);
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.cfg.framer,
      kv: this.createKVClient(payload.key),
      labelClient: this.cfg.labels,
      rangeClient: this,
    });
  }

  sugarMany(payloads: Payload[]): Range[] {
    return payloads.map((payload) => this.sugarOne(payload));
  }
}
