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
} from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { type channel } from "@/channel";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { alias } from "@/ranger/alias";
import { type Client as AliasClient } from "@/ranger/alias/client";
import { kv } from "@/ranger/kv";
import { type Client as KVClient } from "@/ranger/kv/client";
import { type Name, type Params } from "@/ranger/payload";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Payload,
  payloadZ,
} from "@/ranger/types.gen";
import { type Writer } from "@/ranger/writer";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_range_set";
export const DELETE_CHANNEL_NAME = "sy_range_delete";

export const STORE_KEY = "ranges";

const kvDeleteZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

const aliasDeleteZ = z.string().transform((val) => alias.decodeDeleteChange(val));

/** Registers the range, range KV, and range alias stores on the given engine. */
const bindStores = (
  engine: cache.Engine,
  client: Client,
  refetch: (keys: Key[]) => Promise<Range[]>,
): void => {
  const ranges = () => engine.store<Key, Range>(STORE_KEY);
  // Labels and parents are composed from the relationship stores on read, so
  // the event only carries the base payload; enriched fields are preserved.
  const setListener: cache.ChannelListener<{}, typeof payloadZ> = {
    channel: SET_CHANNEL_NAME,
    schema: payloadZ,
    onChange: ({ changed }) =>
      ranges().set(changed.key, (p) =>
        client.sugarOne({ ...changed, labels: p?.labels, parent: p?.parent }),
      ),
  };
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => ranges().delete(changed),
  };
  engine.registerStore<Key, Range>(STORE_KEY, {
    equal: (a, b) => deep.equal(a.payload, b.payload),
    listeners: [setListener, deleteListener],
    refetch,
  });

  const pairs = () => engine.store<string, kv.Pair>(kv.STORE_KEY);
  const kvSetListener: cache.ChannelListener<{}, typeof kv.pairZ> = {
    channel: kv.SET_CHANNEL_NAME,
    schema: kv.pairZ,
    onChange: ({ changed }) => pairs().set(kv.createPairKey(changed), changed),
  };
  const kvDeleteListener: cache.ChannelListener<{}, typeof kvDeleteZ> = {
    channel: kv.DELETE_CHANNEL_NAME,
    schema: kvDeleteZ,
    onChange: ({ changed }) => pairs().delete(kv.createPairKey(changed)),
  };
  engine.registerStore<string, kv.Pair>(kv.STORE_KEY, {
    listeners: [kvSetListener, kvDeleteListener],
  });

  const aliases = () => engine.store<string, alias.Alias>(alias.STORE_KEY);
  const aliasSetListener: cache.ChannelListener<{}, typeof alias.aliasZ> = {
    channel: alias.SET_CHANNEL_NAME,
    schema: alias.aliasZ,
    onChange: ({ changed }) => aliases().set(alias.createKey(changed), changed),
  };
  const aliasDeleteListener: cache.ChannelListener<{}, typeof aliasDeleteZ> = {
    channel: alias.DELETE_CHANNEL_NAME,
    schema: aliasDeleteZ,
    onChange: ({ changed }) => aliases().delete(alias.createKey(changed)),
  };
  engine.registerStore<string, alias.Alias>(alias.STORE_KEY, {
    listeners: [aliasSetListener, aliasDeleteListener],
  });
};

interface RangeConstructionOptions {
  frameClient: framer.Client;
  kv: KVClient;
  aliaser: AliasClient;
  channels: channel.Retriever;
  labelClient: label.Client;
  ontologyClient: ontology.Client;
  rangeClient: Client;
}

export class Range {
  key: string;
  name: string;
  readonly kv: KVClient;
  readonly timeRange: TimeRange;
  readonly color?: color.Color;
  readonly parent?: Payload;
  readonly labels?: label.Label[];
  readonly channels: channel.Retriever;
  private readonly aliaser: AliasClient;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: Client;

  constructor(
    { name, timeRange = TimeRange.ZERO, key, color: color_, parent, labels }: Payload,
    {
      frameClient,
      kv,
      aliaser,
      channels,
      labelClient,
      ontologyClient,
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
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
  }

  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }

  get payload(): Payload {
    const r: Payload = {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
      labels: this.labels,
    };
    if (this.parent != null)
      if ("payload" in this.parent) r.parent = (this.parent as Range).payload;
      else r.parent = this.parent;
    return r;
  }

  async setAlias(channel: channel.Key | Name, alias: string): Promise<void> {
    const ch = await this.channels.retrieve(channel);
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
    return await this.rangeClient.retrieveParent(this.key);
  }

  async retrieveChildren(): Promise<Range[]> {
    const res = (
      await this.ontologyClient.retrieveChildren(this.ontologyID, {
        excludeFieldData: true,
        types: ["range"],
      })
    ).map((r) => r.id.key);
    return await this.rangeClient.retrieve(res);
  }

  async read(channel: Key | Name): Promise<Series>;
  async read(channels: Params): Promise<framer.Frame>;
  async read(channels: Params): Promise<Series | framer.Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }

  async retrieveLabels(): Promise<label.Label[]> {
    return await this.labelClient.retrieve({ for: ontologyID(this.key) });
  }

  async addLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.label(ontologyID(this.key), labels);
  }

  async removeLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.remove(ontologyID(this.key), labels);
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
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeLabels: z.boolean().optional(),
  includeParent: z.boolean().optional(),
});

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

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const retrieveResZ = z.object({ ranges: payloadZ.array().default(() => []) });

/** Scope query-mount subscriptions listen under; see rangeEvents. */
const MOUNT_SCOPE = "range.mounts";

/** The base flags applied to every composed range fetch. */
const BASE_REQUEST: Partial<RetrieveRequest> = {
  includeLabels: true,
  includeParent: true,
};

/** Normalizes non-single retrieve params into a canonical, hashable request. */
const normalizeRequest = (
  params: Exclude<RetrieveParams, Key | Name>,
): RetrieveRequest => retrieveParamsZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.searchTerm == null &&
  req.overlapsWith == null &&
  req.hasLabels == null &&
  req.limit == null &&
  req.offset == null;

const isParentChange = (rel: ontology.Relationship, id: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: id,
  });

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key sets and label filters, permissive for server-computed shapes (search,
 * overlap), which accept every change and drift toward the server's answer.
 */
const requestFilter = (req: RetrieveRequest): ((r: Range) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const labelSet = primitive.isNonZero(req.hasLabels)
    ? new Set(req.hasLabels)
    : undefined;
  return (r) => {
    if (keySet != null && !keySet.has(r.key)) return false;
    if (labelSet != null && !(r.labels ?? []).some((l) => labelSet.has(l.key)))
      return false;
    return true;
  };
};

export class Client {
  readonly type: string = "range";
  private readonly frameClient: framer.Client;
  private readonly writer: Writer;
  private readonly unaryClient: UnaryClient;
  private readonly channels: channel.Retriever;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly createAliasClient: (key: Key) => AliasClient;
  private readonly createKVClient: (key: Key) => KVClient;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key | Name, Range>;
    request: cache.Queries<RetrieveRequest, Range[]>;
    children: cache.Queries<Key, Range[]>;
    parent: cache.Queries<ontology.ID, Range | null>;
    kv: cache.Queries<Key, kv.Pair[]>;
  };

  constructor(
    frameClient: framer.Client,
    writer: Writer,
    unary: UnaryClient,
    channels: channel.Retriever,
    labelClient: label.Client,
    ontologyClient: ontology.Client,
    createAliasClient: (key: Key) => AliasClient,
    createKVClient: (key: Key) => KVClient,
    engine?: cache.Engine,
  ) {
    this.frameClient = frameClient;
    this.writer = writer;
    this.unaryClient = unary;
    this.channels = channels;
    this.labelClient = labelClient;
    this.ontologyClient = ontologyClient;
    this.createAliasClient = createAliasClient;
    this.createKVClient = createKVClient;
    if (engine == null) return;
    bindStores(engine, this, async (keys) => await this.execRetrieve(keys));
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "range",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "ranges",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
      children: new cache.Queries({
        name: "child ranges",
        fetch: async (query) => await this.fetchChildren(query),
        mount: (params) => this.mountChildren(params),
        ensureStreaming,
      }),
      parent: new cache.Queries({
        name: "parent range",
        fetch: async (query) => await this.fetchParent(query),
        mount: (params) => this.mountParent(params),
        ensureStreaming,
      }),
      kv: new cache.Queries({
        name: "range metadata",
        fetch: async (query) => await this.fetchKV(query),
        mount: (params) => this.mountKV(params),
        ensureStreaming,
      }),
    };
  }

  async create(range: New): Promise<Range>;
  async create(ranges: New[]): Promise<Range[]>;
  async create(ranges: New | New[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const news = array.toArray(ranges);
    const res = this.sugarMany(await this.writer.create(news));
    if (this.engine_ != null)
      res.forEach((r, i) => {
        this.writeThrough(r);
        const parent = news[i]?.parent;
        if (r.parent != null || parent == null) return;
        const rel: ontology.Relationship = {
          from: ontologyID(parent.key),
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: ontologyID(r.key),
        };
        this.relationshipStore.set(ontology.relationshipToString(rel), rel);
      });
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name, opts: cache.WriteOptions = {}): Promise<void> {
    const rename = () =>
      this.rangeStore.set(key, (p) =>
        p == null ? undefined : this.sugarOne({ ...p.payload, name }),
      );
    const rollback = new cache.Rollback();
    if (this.engine_ != null) {
      rollback.add(rename());
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(async () => await this.writer.rename(key, name));
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic write while the send was in flight.
    if (this.engine_ != null) rename();
  }

  async delete(key: Key | Key[]): Promise<void> {
    const keys = array.toArray(key);
    await this.writer.delete(keys);
    if (this.engine_ != null) this.rangeStore.delete(keys);
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(params: Key[] | Name[]): Promise<Range[]>;
  async retrieve(params: CrudeTimeRange): Promise<Range[]>;
  async retrieve(params: RetrieveRequest): Promise<Range[]>;
  async retrieve(params: RetrieveParams): Promise<Range | Range[]> {
    const isSingle = typeof params === "string";
    if (this.queries_ == null) {
      const ranges = await this.execRetrieve(params);
      checkForMultipleOrNoResults("Range", params, ranges, isSingle);
      return isSingle ? ranges[0] : ranges;
    }
    if (isSingle) return await this.queries_.single.retrieve(params);
    return await this.queries_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a range; every other shape delivers the matching ranges.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: Key | Name,
    handler: cache.ChangeHandler<Range>,
  ): destructor.Destructor;
  onChange(
    params: Key[] | Name[] | CrudeTimeRange | RetrieveRequest,
    handler: cache.ChangeHandler<Range[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Range> | cache.ChangeHandler<Range[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if (typeof params === "string")
      return queries.single.onChange(params, handler as cache.ChangeHandler<Range>);
    return queries.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Range[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: Key | Name): cache.Cached<Range> | undefined;
  getCached(
    params: Key[] | Name[] | CrudeTimeRange | RetrieveRequest,
  ): cache.Cached<Range[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Range> | cache.Cached<Range[]> | undefined {
    const queries = this.requireQueries();
    if (typeof params === "string") return queries.single.getCached(params);
    return queries.request.getCached(normalizeRequest(params));
  }

  /**
   * Cached queries for the children of a range, keyed by the parent's key.
   * @throws when the cache was disabled at client construction.
   */
  get children(): cache.Queries<Key, Range[]> {
    return this.requireQueries().children;
  }

  /**
   * Cached queries for the closest range parent of a resource, keyed by the
   * child's ontology ID.
   * @throws when the cache was disabled at client construction.
   */
  get parent(): cache.Queries<ontology.ID, Range | null> {
    return this.requireQueries().parent;
  }

  /**
   * Cached queries for a range's KV metadata pairs, keyed by the range's key.
   * @throws when the cache was disabled at client construction.
   */
  get kv(): cache.Queries<Key, kv.Pair[]> {
    return this.requireQueries().kv;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
  }

  private async execRetrieve(params: RetrieveParams): Promise<Range[]> {
    const { ranges } = await this.unaryClient.send(
      "/range/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return this.sugarMany(ranges);
  }

  private get labelStore(): cache.UnaryStore<label.Key, label.Label> {
    return this.requireEngine().store(label.STORE_KEY);
  }

  private get relationshipStore(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get rangeStore(): cache.UnaryStore<Key, Range> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  /** Subscribes to every range set delivered to the cache. */
  onSet(handler: (range: Range) => void): destructor.Destructor {
    return this.rangeEvents.onSet(handler);
  }

  /** Subscribes to every range delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.rangeEvents.onDelete(handler);
  }

  private get rangeEvents(): cache.UnaryStore<Key, Range> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get relationshipEvents(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY, MOUNT_SCOPE);
  }

  private get kvPairStore(): cache.UnaryStore<string, kv.Pair> {
    return this.requireEngine().store(kv.STORE_KEY);
  }

  private get kvPairEvents(): cache.UnaryStore<string, kv.Pair> {
    return this.requireEngine().store(kv.STORE_KEY, MOUNT_SCOPE);
  }

  private get aliasWrites(): cache.UnaryStore<string, alias.Alias> | undefined {
    return this.engine_?.store(alias.STORE_KEY);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  /** Rebuilds a cached range with its cached labels and parent attached. */
  private compose(cached: Range): Range {
    const id = ontologyID(cached.key);
    const labels = label.cachedLabelsOf(this.relationshipStore, this.labelStore, id);
    const next: Payload = { ...cached.payload, labels };
    const parentID = ontology.cachedParentID(this.relationshipStore, id);
    if (parentID == null) delete next.parent;
    else {
      const parent = this.rangeStore.get(parentID.key);
      if (parent != null) next.parent = parent.payload;
    }
    return this.sugarOne(next);
  }

  /** Writes a fetched range and its included labels/parent relationships. */
  private writeThrough(range: Range): void {
    this.rangeStore.set(range.key, range);
    const id = ontologyID(range.key);
    if (range.labels != null) {
      this.labelStore.set(range.labels);
      range.labels.forEach((l) => {
        const rel: ontology.Relationship = {
          from: id,
          type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
          to: label.ontologyID(l.key),
        };
        this.relationshipStore.set(ontology.relationshipToString(rel), rel);
      });
    }
    if (range.parent != null) {
      const rel: ontology.Relationship = {
        from: ontologyID(range.parent.key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: id,
      };
      this.relationshipStore.set(ontology.relationshipToString(rel), rel);
    }
  }

  /**
   * Fetches the given keys, serving composed cached entries and fetching only
   * the misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Range[]> {
    const results: Range[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.rangeStore.get(key);
      if (cached != null) results.push(this.compose(cached));
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ ...BASE_REQUEST, keys: misses });
      fetched.forEach((r) => this.writeThrough(r));
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (r) => r.key);
  }

  private async fetchSingle(query: Key | Name): Promise<Range> {
    const cached = this.rangeStore.get(query);
    if (cached != null) return this.compose(cached);
    const req = keyZ.safeParse(query).success ? { keys: [query] } : { names: [query] };
    const ranges = await this.execRetrieve({ ...BASE_REQUEST, ...req });
    checkForMultipleOrNoResults("Range", query, ranges, true);
    this.writeThrough(ranges[0]);
    return ranges[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key | Name, Range>) {
    const matches = (r: Range) => r.key === query || r.name === query;
    return [
      this.rangeEvents.onSet((range) => {
        if (!matches(range)) return;
        update(() => this.compose(range));
      }),
      this.rangeEvents.onDelete((key) => {
        const corpse = this.rangeStore.getTombstone(key)?.corpse;
        if (key === query || (corpse != null && corpse.name === query)) remove(corpse);
      }),
      this.relationshipEvents.onSet((rel) => {
        update((prev) => {
          if (prev == null) return prev;
          const id = ontologyID(prev.key);
          if (label.matchLabeledBy(rel, id) || isParentChange(rel, id))
            return this.compose(prev);
          return prev;
        });
        void this.ensureRelationshipTargets(rel);
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        update((prev) => {
          if (prev == null) return prev;
          const id = ontologyID(prev.key);
          if (label.matchLabeledBy(rel, id))
            return this.sugarOne({
              ...prev.payload,
              labels: prev.labels?.filter((l) => l.key !== rel.to.key),
            });
          if (isParentChange(rel, id)) return this.compose(prev);
          return prev;
        });
      }),
    ];
  }

  /**
   * Fetches records a relationship points at that the cache is missing, so a
   * subsequent composition can include them. Returns whether anything was
   * fetched, so callers can recompose once the target arrives.
   */
  private async ensureRelationshipTargets(
    rel: ontology.Relationship,
  ): Promise<boolean> {
    if (rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE) {
      if (rel.to.type !== "label" || this.labelStore.has(rel.to.key)) return false;
      const fetched = await this.labelClient.retrieve({ key: rel.to.key });
      this.labelStore.set(rel.to.key, fetched);
      return true;
    }
    if (rel.type === ontology.PARENT_OF_RELATIONSHIP_TYPE) {
      if (rel.from.type !== "range" || this.rangeStore.has(rel.from.key)) return false;
      await this.fetchKeys([rel.from.key]);
      return true;
    }
    return false;
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Range[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const ranges = await this.execRetrieve({ ...BASE_REQUEST, ...query });
    ranges.forEach((r) => this.writeThrough(r));
    return ranges;
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Range[]>) {
    const matches = requestFilter(query);
    return [
      this.rangeEvents.onSet((range) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.find((r) => r.key === range.key);
          const merged = this.compose(range);
          if (!matches(merged))
            return existing == null ? prev : prev.filter((r) => r.key !== range.key);
          if (existing != null)
            return prev.map((r) => (r.key === range.key ? merged : r));
          return [...prev, merged];
        });
      }),
      this.rangeEvents.onDelete((key) => {
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        const apply = () =>
          update((prev) => this.applyRelationship(prev, rel, matches));
        apply();
        void this.ensureRelationshipTargets(rel).then((fetched) => {
          if (fetched) apply();
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        update((prev) => this.applyRelationship(prev, rel, matches));
      }),
    ];
  }

  /**
   * Applies a relationship change to a request answer: recomposes the
   * affected member, drops it when it stops matching, and admits a stored
   * range that starts matching (e.g. a hasLabels query gaining a member).
   */
  private applyRelationship(
    prev: Range[] | undefined,
    rel: ontology.Relationship,
    matches: (r: Range) => boolean,
  ): Range[] | undefined {
    if (prev == null) return prev;
    const isLabelChange =
      rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
      rel.from.type === "range";
    if (!isLabelChange) return this.recomposeAffected(prev, rel);
    const key = rel.from.key;
    const member = prev.find((r) => r.key === key);
    if (member != null) {
      const composed = this.compose(member);
      if (!matches(composed)) return prev.filter((r) => r.key !== key);
      return prev.map((r) => (r.key === key ? composed : r));
    }
    const stored = this.rangeStore.get(key);
    if (stored == null) return prev;
    const composed = this.compose(stored);
    if (!matches(composed)) return prev;
    return [...prev, composed];
  }

  /** Recomposes the members of prev whose labels or parent the rel affects. */
  private recomposeAffected(
    prev: Range[] | undefined,
    rel: ontology.Relationship,
  ): Range[] | undefined {
    if (prev == null) return prev;
    return prev.map((r) => {
      const id = ontologyID(r.key);
      if (label.matchLabeledBy(rel, id) || isParentChange(rel, id))
        return this.compose(r);
      return r;
    });
  }

  private async fetchChildren(query: Key): Promise<Range[]> {
    const resources = await this.ontologyClient.retrieveChildren(ontologyID(query), {
      types: ["range"],
    });
    if (resources.length === 0) return [];
    return await this.fetchKeys(resources.map(({ id: { key } }) => key));
  }

  private mountChildren({ query, update }: cache.MountParams<Key, Range[]>) {
    const parentID = ontologyID(query);
    return [
      this.rangeEvents.onSet((range) => {
        update((prev) => {
          if (prev == null || !prev.some((r) => r.key === range.key)) return prev;
          return prev.map((r) => (r.key === range.key ? this.compose(range) : r));
        });
      }),
      this.rangeEvents.onDelete((key) => {
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        const isNewChild = ontology.matchRelationship(rel, {
          from: parentID,
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: { type: "range" },
        });
        if (!isNewChild) {
          update((prev) => this.recomposeAffected(prev, rel));
          void this.ensureRelationshipTargets(rel);
          return;
        }
        void this.fetchKeys([rel.to.key]).then(([child]) => {
          if (child == null) return;
          update((prev) => {
            if (prev == null) return prev;
            if (prev.some((r) => r.key === child.key))
              return prev.map((r) => (r.key === child.key ? child : r));
            return [...prev, child];
          });
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        const isChild = ontology.matchRelationship(rel, {
          from: parentID,
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: { type: "range" },
        });
        if (isChild) update((prev) => prev?.filter((r) => r.key !== rel.to.key));
        else update((prev) => this.recomposeAffected(prev, rel));
      }),
    ];
  }

  private async fetchParent(query: ontology.ID): Promise<Range | null> {
    const res = await this.ontologyClient.retrieveParents(query);
    const parent = res.find(({ id: { type } }) => type === "range");
    if (parent == null) return null;
    const [range] = await this.fetchKeys([parent.id.key]);
    return range ?? null;
  }

  private mountParent({
    query: childID,
    update,
  }: cache.MountParams<ontology.ID, Range | null>) {
    return [
      this.rangeEvents.onSet((range) => {
        update((prev) =>
          prev == null || prev.key !== range.key ? prev : this.compose(range),
        );
      }),
      this.relationshipEvents.onSet((rel) => {
        const isParent = ontology.matchRelationship(rel, {
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: childID,
        });
        if (!isParent) return;
        if (rel.from.type !== "range") return update(null);
        void this.fetchKeys([rel.from.key]).then(([parent]) => {
          if (parent != null) update(parent);
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        const isParent = ontology.matchRelationship(rel, {
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: childID,
        });
        if (isParent) update(null);
      }),
    ];
  }

  private async fetchKV(query: Key): Promise<kv.Pair[]> {
    const pairs = await this.createKVClient(query).list();
    const result: kv.Pair[] = Object.entries(pairs).map(([key, value]) => ({
      range: query,
      key,
      value,
    }));
    result.forEach((p) => this.kvPairStore.set(kv.createPairKey(p), p));
    return result;
  }

  private mountKV({ query, update }: cache.MountParams<Key, kv.Pair[]>) {
    return [
      this.kvPairEvents.onSet((pair) => {
        if (pair.range !== query) return;
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.find((p) => p.key === pair.key);
          if (existing == null) return [...prev, pair];
          return prev.map((p) => (p.key === pair.key ? pair : p));
        });
      }),
      this.kvPairEvents.onDelete((pairKey) => {
        const { range, key } = kvDeleteZ.parse(pairKey);
        if (range !== query) return;
        update((prev) => prev?.filter((p) => p.key !== key));
      }),
    ];
  }

  getKV(range: Key): KVClient {
    return this.createKVClient(range);
  }

  /** Returns the parent of the given range, or null when it has none. */
  async retrieveParent(range: Key): Promise<Range | null> {
    if (this.queries_ != null)
      return await this.queries_.parent.retrieve(ontologyID(range));
    const res = await this.ontologyClient.retrieveParents(ontologyID(range));
    if (res.length === 0) return null;
    const first = res[0];
    if (first.id.type !== "range") return null;
    return await this.retrieve(first.id.key);
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

  async setAlias(range: Key, channel: channel.Key, aliasName: string): Promise<void> {
    await this.createAliasClient(range).set({ [channel]: aliasName });
    const entry: alias.Alias = { range, channel, alias: aliasName };
    this.aliasWrites?.set(alias.createKey(entry), entry);
  }

  async deleteAlias(range: Key, channels: channel.Key | channel.Key[]): Promise<void> {
    const channelsArr = array.toArray(channels);
    await this.createAliasClient(range).delete(channelsArr);
    this.aliasWrites?.delete(
      channelsArr.map((channel) => alias.createKey({ range, channel })),
    );
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.frameClient,
      kv: this.createKVClient(payload.key),
      aliaser: this.createAliasClient(payload.key),
      channels: this.channels,
      labelClient: this.labelClient,
      ontologyClient: this.ontologyClient,
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

export const aliasOntologyID = (key: Key): ontology.ID => ({
  type: "range-alias",
  key,
});

export const convertOntologyResourceToPayload = ({
  data,
  id: { key },
  name,
}: ontology.Resource): Payload => {
  const timeRange = TimeRange.z.parse(data?.timeRange);
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
