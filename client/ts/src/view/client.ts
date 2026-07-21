// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { bindStore, STORE_KEY } from "@/view/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type View,
  viewZ,
} from "@/view/types.gen";

const createReqZ = z.object({ views: viewZ.array() });
const createResZ = z.object({ views: viewZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  types: z.string().array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export interface RetrieveSingleParams extends z.input<typeof singleRetrieveParamsZ> {}
export interface RetrieveMultipleParams extends z.input<typeof retrieveRequestZ> {}

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResponseZ = z.object({ views: viewZ.array().default(() => []) });

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.types == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side matching for a request: key and type sets. Server-computed
 * shapes (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((v: View) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const typeSet = primitive.isNonZero(req.types) ? new Set(req.types) : undefined;
  return (v) => {
    if (keySet != null && !keySet.has(v.key)) return false;
    if (typeSet != null && !typeSet.has(v.type)) return false;
    return true;
  };
};

export class Client {
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key, View, Key, View>;
    request: cache.Answers<RetrieveRequest, View[], Key, View>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "view",
        table: this.viewStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((v) => v.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "views",
        table: this.viewStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((v) => v.key),
        compose: (records) => records,
        matches: (view, query) => requestFilter(query)(view),
        serverFields: SERVER_FIELDS,
      }),
    };
  }

  async retrieve(params: RetrieveSingleParams): Promise<View>;
  async retrieve(params: RetrieveMultipleParams): Promise<View[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<View | View[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(retrieveRequestZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a view; every other shape delivers the matching views.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<View>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<View[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<View> | cache.ChangeHandler<View[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(params.key, handler as cache.ChangeHandler<View>);
    return answers.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<View[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<View> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<View[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<View> | cache.Cached<View[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(retrieveRequestZ.parse(params));
  }

  async create(view: New): Promise<View>;
  async create(views: New[]): Promise<View[]>;
  async create(views: New | New[]): Promise<View | View[]> {
    const isMany = Array.isArray(views);
    const res = await this.client.send(
      "/view/create",
      { views: array.toArray(views) },
      createReqZ,
      createResZ,
    );
    this.writes.setMany(res.views);
    return isMany ? res.views : res.views[0];
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const v = await this.retrieve({ key });
    const rollback = new cache.Rollback();
    const writes = this.writes;
    rollback.add(cache.partialUpdate(writes, key, { name }));
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create({ ...v, name });
    });
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    const ids = ontologyID(keysArr);
    rollback.add(ontology.deleteCachedRelationships(this.cache_, ids));
    rollback.add(writes.delete(keysArr));
    const resources = this.cache_.table<string, ontology.Resource>(
      ontology.RESOURCES_STORE_KEY,
    );
    rollback.add(resources.delete(ontology.idToString(ids)));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/view/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    this.writes.delete(keysArr);
  }

  private get writes(): cache.Table<Key, View> {
    return this.cache_.table(STORE_KEY);
  }

  private get viewStore(): cache.Table<Key, View> {
    return this.cache_.table(STORE_KEY);
  }

  /** Subscribes to every view set delivered to the cache. */
  onSet(handler: (view: View) => void): destructor.Destructor {
    return this.viewStore.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every view delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.viewStore.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private async execRetrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<View[]> {
    const res = await this.client.send(
      "/view/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.views;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<View[]> {
    const results: View[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.viewStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.viewStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (v) => v.key);
  }

  private async fetchSingle(query: Key): Promise<View> {
    const cached = this.viewStore.get(query);
    if (cached != null) return cached;
    const views = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("View", query, views, true);
    this.viewStore.setMany(views);
    return views[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<View[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const views = await this.execRetrieve(query);
    this.viewStore.setMany(views);
    return views;
  }
}
