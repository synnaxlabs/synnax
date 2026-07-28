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
  destructor,
  primitive,
  type Series,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { alias } from "@/ranger/alias";
import { Client as AliasClient } from "@/ranger/alias/client";
import { kv } from "@/ranger/kv";
import { Client as KVClient } from "@/ranger/kv/client";
import { type Name, type Params } from "@/ranger/payload";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Payload,
  payloadZ,
} from "@/ranger/types.gen";
import { Writer } from "@/ranger/writer";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_range_set";
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
  sugarOne: (payload: Payload) => Range,
  relationships: query.Table<string, ontology.Relationship>,
  fetch: (keys: Key[]) => Promise<Range[]>,
  backfill: (rel: ontology.Relationship) => Promise<void>,
): Stores => {
  const ranges = cache.createTable<Key, Range>({
    name: "ranges",
    equal: (a, b) => deep.equal(a.payload, b.payload),
    fetch,
    listen: [
      // Labels and parents are composed from the relationship tables on read,
      // so the event only carries the base payload; enriched fields are
      // preserved.
      query.createSetListener(SET_CHANNEL_NAME, payloadZ, {
        value: (changed, prev) =>
          sugarOne({ ...changed, labels: prev?.labels, parent: prev?.parent }),
      }),
      query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
    ],
  });

  // Fetches missing relationship targets so compositions and membership
  // checks can see them.
  relationships.subscribe((event) => {
    if (event.variant === "set") void backfill(event.value).catch(cache.onError);
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
      await this.ontologyClient.children.retrieve({
        ids: this.ontologyID,
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
  ignoreNotFoundError: z.boolean().optional(),
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

/** The base flags applied to every composed range fetch. */
const BASE_REQUEST: Partial<RetrieveRequest> = {
  includeLabels: true,
  includeParent: true,
};

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
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
  event: query.TableEvent<string, ontology.Relationship>,
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

/** Projects relationship events onto the range keys they affect. */
const watchRelationships = <Q extends query.Params>(
  relationships: query.Table<string, ontology.Relationship>,
): query.WatchEntry<Q, Key> =>
  query.watch<Q, Key, string, ontology.Relationship>(relationships, (event) =>
    affectedRangeKeys(relOfEvent(event)),
  );

const rangesWithLabel = (
  relationships: query.Table<string, ontology.Relationship>,
  key: label.Key,
): Key[] | null => {
  const keys = relationships
    .get(
      (r) =>
        r.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
        r.from.type === "range" &&
        r.to.key === key,
    )
    .map((r) => r.from.key);
  return keys.length === 0 ? null : keys;
};

/** Projects label content changes onto the ranges they label. */
const watchLabels = <Q extends query.Params>(
  labels: query.Table<label.Key, label.Label>,
  relationships: query.Table<string, ontology.Relationship>,
): query.WatchEntry<Q, Key> =>
  query.watch<Q, Key, label.Key, label.Label>(labels, (event) =>
    rangesWithLabel(relationships, event.key),
  );

export interface ClientConfig {
  framer: framer.Client;
  unary: UnaryClient;
  channels: channel.Retriever;
  labels: label.Client;
  ontology: ontology.Client;
  cache: query.Cache;
}

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
  readonly children: query.Retrieves<Key, Range[]>;
  /**
   * Cached queries for the closest range parent of a resource, keyed by the
   * child's ontology ID.
   */
  readonly parent: query.Retrieves<ontology.ID, Range | null>;
  /** Cached queries for a range's KV metadata pairs, keyed by the range's key. */
  readonly kv: query.Retrieves<Key, kv.Pair[]>;
  private readonly cfg: ClientConfig;
  private readonly writer: Writer;
  private readonly store: query.Table<Key, Range>;
  private readonly kvPairs: query.Table<string, kv.Pair>;

  constructor(cfg: ClientConfig) {
    const { labels: labelClient, ontology: ontologyClient, cache } = cfg;
    const labels = labelClient.store;
    const { relationships } = ontologyClient.cache;
    const { ranges, kvPairs, aliases } = createTables(
      cache,
      (payload) => this.sugarOne(payload),
      relationships,
      async (keys) => await this.fetchThrough(keys),
      async (rel) => await this.ensureRelationshipTargets(rel),
    );
    const single = cache.queries<Key | Name, Range, Key, Range>({
      name: "range",
      table: ranges,
      fetch: async (query) => [(await this.fetchSingle(query)).key],
      compose: ([record]) => this.composeOne(record),
      keyOf: (query) => (keyZ.safeParse(query).success ? query : null),
      matches: (r, query) => r.key === query || r.name === query,
      single: true,
      watch: [
        watchRelationships<Key | Name>(relationships),
        watchLabels<Key | Name>(labels, relationships),
      ],
    });
    super(cache, {
      name: "range",
      table: ranges,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (query) => await this.fetchRequest(query),
        matches: (r, query) => this.requestMatches(r, query),
        watch: [
          watchRelationships<RetrieveRequest>(relationships),
          watchLabels<RetrieveRequest>(labels, relationships),
        ],
      },
      compose: (r) => this.composeOne(r),
      single: {
        is: (params) => typeof params === "string",
        normalize: (params) => params,
        space: single as query.Retrieves<query.Params, Range>,
      },
    });
    this.cfg = cfg;
    this.writer = new Writer(cfg.unary);
    this.store = ranges;
    this.kvPairs = kvPairs;
    this.aliases = aliases;
    this.children = cache.queries<Key, Range[], Key, Range>({
      name: "child ranges",
      table: this.store,
      fetch: async (query) => (await this.fetchChildren(query)).map((r) => r.key),
      compose: (records) => records.map((r) => this.composeOne(r)),
      matches: (r, query) => {
        const parent = this.cfg.ontology.cache.parentID(ontologyID(r.key));
        return parent != null && ontology.idsEqual(parent, ontologyID(query));
      },
      watch: [
        watchRelationships<Key>(this.cfg.ontology.cache.relationships),
        watchLabels<Key>(this.cfg.labels.store, this.cfg.ontology.cache.relationships),
      ],
    });
    this.parent = cache.queries<ontology.ID, Range | null, Key, Range>({
      name: "parent range",
      table: this.store,
      fetch: async (query) => {
        const range = await this.fetchParent(query);
        return range == null ? [] : [range.key];
      },
      compose: (records) => (records[0] == null ? null : this.composeOne(records[0])),
      matches: (r, query) => {
        const parent = this.cfg.ontology.cache.parentID(query);
        return parent != null && parent.type === "range" && parent.key === r.key;
      },
      watch: [
        query.watch<ontology.ID, Key, string, ontology.Relationship>(
          this.cfg.ontology.cache.relationships,
          (event, query) => {
            const rel = relOfEvent(event);
            if (isParentChange(rel, query))
              return rel.from.type === "range" ? [rel.from.key] : "refetch";
            return affectedRangeKeys(rel);
          },
        ),
        watchLabels<ontology.ID>(
          this.cfg.labels.store,
          this.cfg.ontology.cache.relationships,
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
      this.cfg.ontology.cache.relationships.set(
        ontology.relationshipToString(rel),
        rel,
      );
    });
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      this.store.set(key, (p) =>
        p == null ? undefined : this.sugarOne({ ...p.payload, name }),
      ),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    const rollbacks = new destructor.Chain();
    rollbacks.add(...rename());
    await opts.onOptimistic?.();
    await rollbacks.guard(async () => await this.writer.rename(key, name));
    rename();
  }

  async delete(key: Key | Key[]): Promise<void> {
    const keys = array.toArray(key);
    await this.writer.delete(keys);
    this.store.delete(keys);
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(params: Key[] | Name[]): Promise<Range[]>;
  async retrieve(params: CrudeTimeRange): Promise<Range[]>;
  async retrieve(params: RetrieveRequest): Promise<Range[]>;
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

  /** Rebuilds a cached range with its cached labels and parent attached. */
  private composeOne(cached: Range): Range {
    const id = ontologyID(cached.key);
    const labels = label.cachedLabelsOf(
      this.cfg.ontology.cache.relationships,
      this.cfg.labels.store,
      id,
    );
    const next: Payload = { ...cached.payload, labels };
    const parentID = this.cfg.ontology.cache.parentID(id);
    if (parentID == null) delete next.parent;
    else {
      const parent = this.store.get(parentID.key);
      if (parent != null) next.parent = parent.payload;
    }
    return this.sugarOne(next);
  }

  /** Writes a fetched range and its included labels/parent relationships. */
  private writeThrough(range: Range): void {
    this.store.set(range);
    const id = ontologyID(range.key);
    if (range.labels != null) {
      this.cfg.labels.store.set(range.labels);
      range.labels.forEach((l) => {
        const rel: ontology.Relationship = {
          from: id,
          type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
          to: label.ontologyID(l.key),
        };
        this.cfg.ontology.cache.relationships.set(
          ontology.relationshipToString(rel),
          rel,
        );
      });
    }
    if (range.parent != null) {
      const rel: ontology.Relationship = {
        from: ontologyID(range.parent.key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: id,
      };
      this.cfg.ontology.cache.relationships.set(
        ontology.relationshipToString(rel),
        rel,
      );
    }
  }

  /**
   * Fetches the given keys with labels and parents included, writing them and
   * their relationships through. Powers the table's fetch primitive.
   */
  private async fetchThrough(keys: Key[]): Promise<Range[]> {
    const ranges = await this.execRetrieve({
      ...BASE_REQUEST,
      keys,
      ignoreNotFoundError: true,
    });
    ranges.forEach((r) => this.writeThrough(r));
    return ranges;
  }

  private async fetchSingle(query: Key | Name): Promise<Range> {
    const cached = this.store.get(query);
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
      if (rel.to.type !== "label" || this.cfg.labels.store.has(rel.to.key)) return;
      const fetched = await this.cfg.labels.retrieve({ key: rel.to.key });
      this.cfg.labels.store.set(rel.to.key, fetched);
      return;
    }
    if (rel.type === ontology.PARENT_OF_RELATIONSHIP_TYPE) {
      if (rel.from.type !== "range" || this.store.has(rel.from.key)) return;
      await this.store.retrieve([rel.from.key]);
    }
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Range[]> {
    if (isKeysOnly(query)) return await this.store.retrieve(query.keys);
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
        this.cfg.ontology.cache.relationships,
        this.cfg.labels.store,
        ontologyID(r.key),
      );
      const wanted = new Set(req.hasLabels);
      if (!labels.some((l) => wanted.has(l.key))) return false;
    }
    return true;
  }

  private async fetchChildren(query: Key): Promise<Range[]> {
    const resources = await this.cfg.ontology.children.retrieve({
      ids: ontologyID(query),
      types: ["range"],
    });
    if (resources.length === 0) return [];
    return await this.store.retrieve(resources.map(({ id: { key } }) => key));
  }

  private async fetchParent(query: ontology.ID): Promise<Range | null> {
    const res = await this.cfg.ontology.parents.retrieve({ ids: query });
    const parent = res.find(({ id: { type } }) => type === "range");
    if (parent == null) return null;
    const [range] = await this.store.retrieve([parent.id.key]);
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
    return await this.parent.retrieve(ontologyID(range));
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
    this.aliases.set(alias.createKey(entry), entry);
  }

  async deleteAlias(range: Key, channels: channel.Key | channel.Key[]): Promise<void> {
    const channelsArr = array.toArray(channels);
    await this.createAliasClient(range).delete(channelsArr);
    this.aliases.delete(
      channelsArr.map((channel) => alias.createKey({ range, channel })),
    );
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.cfg.framer,
      kv: this.createKVClient(payload.key),
      aliaser: this.createAliasClient(payload.key),
      channels: this.cfg.channels,
      labelClient: this.cfg.labels,
      ontologyClient: this.cfg.ontology,
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
