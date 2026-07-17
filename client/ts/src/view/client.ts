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

const MOUNT_SCOPE = "view.mounts";

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.types == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key and type sets, permissive for server-computed shapes (search), which
 * accept every change and drift toward the server's answer.
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
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, View>;
    request: cache.Queries<RetrieveRequest, View[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "view",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "views",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  async retrieve(params: RetrieveSingleParams): Promise<View>;
  async retrieve(params: RetrieveMultipleParams): Promise<View[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<View | View[]> {
    const isSingle = "key" in params;
    if (this.queries_ == null) {
      const views = await this.execRetrieve(params);
      checkForMultipleOrNoResults("View", params, views, isSingle);
      return isSingle ? views[0] : views;
    }
    if (isSingle) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.request.retrieve(retrieveRequestZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a view; every other shape delivers the matching views.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(params.key, handler as cache.ChangeHandler<View>);
    return queries.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<View[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<View> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<View[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<View> | cache.Cached<View[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(retrieveRequestZ.parse(params));
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
    this.writes?.set(res.views);
    return isMany ? res.views : res.views[0];
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const v = await this.retrieve({ key });
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create({ ...v, name });
    });
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      const ids = ontologyID(keysArr);
      rollback.add(ontology.deleteCachedRelationships(this.engine_, ids));
      rollback.add(writes.delete(keysArr));
      const resources = this.engine_.store<string, ontology.Resource>(
        ontology.RESOURCES_STORE_KEY,
      );
      rollback.add(resources.delete(ontology.idToString(ids)));
    }
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
    this.writes?.delete(keysArr);
  }

  private get writes(): cache.UnaryStore<Key, View> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get viewStore(): cache.UnaryStore<Key, View> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  /** Subscribes to every view set delivered to the cache. */
  onSet(handler: (view: View) => void): destructor.Destructor {
    return this.viewEvents.onSet(handler);
  }

  /** Subscribes to every view delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.viewEvents.onDelete(handler);
  }

  private get viewEvents(): cache.UnaryStore<Key, View> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
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
      this.viewStore.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (v) => v.key);
  }

  private async fetchSingle(query: Key): Promise<View> {
    const cached = this.viewStore.get(query);
    if (cached != null) return cached;
    const views = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("View", query, views, true);
    this.viewStore.set(views);
    return views[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, View>) {
    return [
      this.viewEvents.onSet((view) => {
        if (view.key === query) update(view);
      }),
      this.viewEvents.onDelete((key) => {
        if (key === query) remove(this.viewStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<View[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const views = await this.execRetrieve(query);
    this.viewStore.set(views);
    return views;
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, View[]>) {
    const matches = requestFilter(query);
    return [
      this.viewEvents.onSet((view) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((v) => v.key === view.key);
          if (!matches(view))
            return existing ? prev.filter((v) => v.key !== view.key) : prev;
          if (existing) return prev.map((v) => (v.key === view.key ? view : v));
          return [...prev, view];
        });
      }),
      this.viewEvents.onDelete((key) => {
        update((prev) => prev?.filter((v) => v.key !== key));
      }),
    ];
  }
}
