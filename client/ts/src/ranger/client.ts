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

/** Registers the range, range KV, and range alias tables on the given cache. */
const bindStores = (
  engine: cache.Cache,
  client: Client,
  refetch: (keys: Key[]) => Promise<Range[]>,
  backfill: (rel: ontology.Relationship) => Promise<void>,
): void => {
  const ranges = () => engine.table<Key, Range>(STORE_KEY);
  // Labels and parents are composed from the relationship tables on read, so
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
  engine.registerTable<Key, Range>(STORE_KEY, {
    equal: (a, b) => deep.equal(a.payload, b.payload),
    listeners: [setListener, deleteListener],
    refetch,
  });

  // Fetches missing relationship targets so compositions and membership
  // checks can see them.
  const relationshipSet: cache.ChannelListener<{}, typeof ontology.relationshipZ> = {
    channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
    schema: ontology.relationshipZ,
    onChange: async ({ changed }) => await backfill(changed),
  };
  engine.addListeners(ontology.RELATIONSHIPS_STORE_KEY, relationshipSet);

  const pairs = () => engine.table<string, kv.Pair>(kv.STORE_KEY);
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
  engine.registerTable<string, kv.Pair>(kv.STORE_KEY, {
    listeners: [kvSetListener, kvDeleteListener],
  });

  const aliases = () => engine.table<string, alias.Alias>(alias.STORE_KEY);
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
  engine.registerTable<string, alias.Alias>(alias.STORE_KEY, {
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

/** The base flags applied to every composed range fetch. */
const BASE_REQUEST: Partial<RetrieveRequest> = {
  includeLabels: true,
  includeParent: true,
};

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

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

const relOfEvent = (
  event: cache.TableEvent<string, ontology.Relationship>,
): ontology.Relationship =>
  event.variant === "set" ? event.value : ontology.relationshipZ.parse(event.key);

/** Range keys whose composed labels or parent the relationship affects. */
const affectedRangeKeys = (rel: ontology.Relationship): Key[] | null => {
  const keys: Key[] = [];
  const labeled = ontology.matchRelationship(rel, {
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
    from: { type: "range" },
  });
  if (labeled) keys.push(rel.from.key);
  const parented = ontology.matchRelationship(rel, {
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "range" },
  });
  if (parented) keys.push(rel.to.key);
  return keys.length === 0 ? null : keys;
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
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key | Name, Range, Key, Range>;
    request: cache.Answers<RetrieveRequest, Range[], Key, Range>;
    children: cache.Answers<Key, Range[], Key, Range>;
    parent: cache.Answers<ontology.ID, Range | null, Key, Range>;
    kv: cache.Answers<Key, kv.Pair[], string, kv.Pair>;
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
    engine: cache.Cache,
  ) {
    this.frameClient = frameClient;
    this.writer = writer;
    this.unaryClient = unary;
    this.channels = channels;
    this.labelClient = labelClient;
    this.ontologyClient = ontologyClient;
    this.createAliasClient = createAliasClient;
    this.createKVClient = createKVClient;
    bindStores(
      engine,
      this,
      async (keys) => await this.execRetrieve(keys),
      async (rel) => await this.ensureRelationshipTargets(rel),
    );
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "range",
        table: this.rangeStore,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: ([record]) => this.composeOne(record),
        keyOf: (query) => (keyZ.safeParse(query).success ? query : null),
        matches: (r, query) => r.key === query || r.name === query,
        single: true,
        watch: [this.watchRelationships<Key | Name>(), this.watchLabels<Key | Name>()],
      }),
      request: engine.answers({
        name: "ranges",
        table: this.rangeStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((r) => r.key),
        compose: (records) => records.map((r) => this.composeOne(r)),
        matches: (r, query) => this.requestMatches(r, query),
        serverFields: SERVER_FIELDS,
        watch: [
          this.watchRelationships<RetrieveRequest>(),
          this.watchLabels<RetrieveRequest>(),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
      children: engine.answers({
        name: "child ranges",
        table: this.rangeStore,
        fetch: async (query) => (await this.fetchChildren(query)).map((r) => r.key),
        compose: (records) => records.map((r) => this.composeOne(r)),
        matches: (r, query) => {
          const parent = ontology.cachedParentID(
            this.relationshipStore,
            ontologyID(r.key),
          );
          return parent != null && ontology.idsEqual(parent, ontologyID(query));
        },
        watch: [this.watchRelationships<Key>(), this.watchLabels<Key>()],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
      parent: engine.answers({
        name: "parent range",
        table: this.rangeStore,
        fetch: async (query) => {
          const range = await this.fetchParent(query);
          return range == null ? [] : [range.key];
        },
        compose: (records) => (records[0] == null ? null : this.composeOne(records[0])),
        matches: (r, query) => {
          const parent = ontology.cachedParentID(this.relationshipStore, query);
          return parent != null && parent.type === "range" && parent.key === r.key;
        },
        watch: [
          cache.watch<ontology.ID, Key, string, ontology.Relationship>(
            this.relationshipStore,
            (event, query) => {
              const rel = relOfEvent(event);
              if (isParentChange(rel, query))
                return rel.from.type === "range" ? [rel.from.key] : "refetch";
              return affectedRangeKeys(rel);
            },
          ),
          this.watchLabels<ontology.ID>(),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
      kv: engine.answers({
        name: "range metadata",
        table: this.kvPairStore,
        fetch: async (query) =>
          (await this.fetchKV(query)).map((p) => kv.createPairKey(p)),
        compose: (records) => records,
        matches: (pair, query) => pair.range === query,
      }),
    };
  }

  async create(range: New): Promise<Range>;
  async create(ranges: New[]): Promise<Range[]>;
  async create(ranges: New | New[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const news = array.toArray(ranges);
    const res = this.sugarMany(await this.writer.create(news));
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
    rollback.add(rename());
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
    await opts.onOptimistic?.();
    await rollback.guard(async () => await this.writer.rename(key, name));
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic write while the send was in flight.
    rename();
  }

  async delete(key: Key | Key[]): Promise<void> {
    const keys = array.toArray(key);
    await this.writer.delete(keys);
    this.rangeStore.delete(keys);
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(params: Key[] | Name[]): Promise<Range[]>;
  async retrieve(params: CrudeTimeRange): Promise<Range[]>;
  async retrieve(params: RetrieveRequest): Promise<Range[]>;
  async retrieve(params: RetrieveParams): Promise<Range | Range[]> {
    const isSingle = typeof params === "string";
    if (isSingle) return await this.answers_.single.retrieve(params);
    return await this.answers_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a range; every other shape delivers the matching ranges.
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
    const answers = this.answers_;
    if (typeof params === "string")
      return answers.single.onChange(params, handler as cache.ChangeHandler<Range>);
    return answers.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Range[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: Key | Name): cache.Cached<Range> | undefined;
  getCached(
    params: Key[] | Name[] | CrudeTimeRange | RetrieveRequest,
  ): cache.Cached<Range[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Range> | cache.Cached<Range[]> | undefined {
    const answers = this.answers_;
    if (typeof params === "string") return answers.single.getCached(params);
    return answers.request.getCached(normalizeRequest(params));
  }

  /** Cached queries for the children of a range, keyed by the parent's key. */
  get children(): cache.Answers<Key, Range[], Key, Range> {
    return this.answers_.children;
  }

  /**
   * Cached queries for the closest range parent of a resource, keyed by the
   * child's ontology ID.
   */
  get parent(): cache.Answers<ontology.ID, Range | null, Key, Range> {
    return this.answers_.parent;
  }

  /** Cached queries for a range's KV metadata pairs, keyed by the range's key. */
  get kv(): cache.Answers<Key, kv.Pair[], string, kv.Pair> {
    return this.answers_.kv;
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

  private get labelStore(): cache.Table<label.Key, label.Label> {
    return this.cache_.table(label.STORE_KEY);
  }

  private get relationshipStore(): cache.Table<string, ontology.Relationship> {
    return this.cache_.table(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get rangeStore(): cache.Table<Key, Range> {
    return this.cache_.table(STORE_KEY);
  }

  /** Subscribes to every range set delivered to the cache. */
  onSet(handler: (range: Range) => void): destructor.Destructor {
    return this.rangeStore.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every range delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.rangeStore.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private get kvPairStore(): cache.Table<string, kv.Pair> {
    return this.cache_.table(kv.STORE_KEY);
  }

  private get aliasWrites(): cache.Table<string, alias.Alias> {
    return this.cache_.table(alias.STORE_KEY);
  }

  /** Projects relationship events onto the range keys they affect. */
  private watchRelationships<Q extends cache.Query>(): cache.WatchEntry<Q, Key> {
    return cache.watch<Q, Key, string, ontology.Relationship>(
      this.relationshipStore,
      (event) => affectedRangeKeys(relOfEvent(event)),
    );
  }

  /** Projects label content changes onto the ranges they label. */
  private watchLabels<Q extends cache.Query>(): cache.WatchEntry<Q, Key> {
    return cache.watch<Q, Key, label.Key, label.Label>(this.labelStore, (event) =>
      this.rangesWithLabel(event.key),
    );
  }

  private rangesWithLabel(key: label.Key): Key[] | null {
    const keys = this.relationshipStore
      .get(
        (r) =>
          r.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
          r.from.type === "range" &&
          r.to.key === key,
      )
      .map((r) => r.from.key);
    return keys.length === 0 ? null : keys;
  }

  /** Rebuilds a cached range with its cached labels and parent attached. */
  private composeOne(cached: Range): Range {
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
      this.labelStore.setMany(range.labels);
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
      if (cached != null) results.push(this.composeOne(cached));
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
    if (cached != null) return this.composeOne(cached);
    const req = keyZ.safeParse(query).success ? { keys: [query] } : { names: [query] };
    const ranges = await this.execRetrieve({ ...BASE_REQUEST, ...req });
    checkForMultipleOrNoResults("Range", query, ranges, true);
    this.writeThrough(ranges[0]);
    return ranges[0];
  }

  /**
   * Fetches records a relationship points at that the cache is missing, so
   * compositions and membership checks can include them. Presence guards make
   * it idempotent.
   */
  private async ensureRelationshipTargets(rel: ontology.Relationship): Promise<void> {
    if (rel.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE) {
      if (rel.to.type !== "label" || this.labelStore.has(rel.to.key)) return;
      const fetched = await this.labelClient.retrieve({ key: rel.to.key });
      this.labelStore.set(rel.to.key, fetched);
      return;
    }
    if (rel.type === ontology.PARENT_OF_RELATIONSHIP_TYPE) {
      if (rel.from.type !== "range" || this.rangeStore.has(rel.from.key)) return;
      await this.fetchKeys([rel.from.key]);
    }
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Range[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const ranges = await this.execRetrieve({ ...BASE_REQUEST, ...query });
    ranges.forEach((r) => this.writeThrough(r));
    return ranges;
  }

  /** Exact client-side evaluation of a request against a cached range. */
  private requestMatches(r: Range, req: RetrieveRequest): boolean {
    if (primitive.isNonZero(req.keys) && !req.keys.includes(r.key)) return false;
    if (primitive.isNonZero(req.names) && !req.names.includes(r.name)) return false;
    if (
      req.overlapsWith != null &&
      !new TimeRange(req.overlapsWith).overlapsWith(r.timeRange)
    )
      return false;
    if (primitive.isNonZero(req.hasLabels)) {
      const labels = label.cachedLabelsOf(
        this.relationshipStore,
        this.labelStore,
        ontologyID(r.key),
      );
      const wanted = new Set(req.hasLabels);
      if (!labels.some((l) => wanted.has(l.key))) return false;
    }
    return true;
  }

  private async fetchChildren(query: Key): Promise<Range[]> {
    const resources = await this.ontologyClient.retrieveChildren(ontologyID(query), {
      types: ["range"],
    });
    if (resources.length === 0) return [];
    return await this.fetchKeys(resources.map(({ id: { key } }) => key));
  }

  private async fetchParent(query: ontology.ID): Promise<Range | null> {
    const res = await this.ontologyClient.retrieveParents(query);
    const parent = res.find(({ id: { type } }) => type === "range");
    if (parent == null) return null;
    const [range] = await this.fetchKeys([parent.id.key]);
    return range ?? null;
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

  getKV(range: Key): KVClient {
    return this.createKVClient(range);
  }

  /** Returns the parent of the given range, or null when it has none. */
  async retrieveParent(range: Key): Promise<Range | null> {
    return await this.answers_.parent.retrieve(ontologyID(range));
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
    this.aliasWrites.set(alias.createKey(entry), entry);
  }

  async deleteAlias(range: Key, channels: channel.Key | channel.Key[]): Promise<void> {
    const channelsArr = array.toArray(channels);
    await this.createAliasClient(range).delete(channelsArr);
    this.aliasWrites.delete(
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
